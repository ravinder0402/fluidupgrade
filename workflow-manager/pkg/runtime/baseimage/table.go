package baseimage

import (
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	pkgerrors "github.com/coredgeio/compass/pkg/errors"
	"github.com/coredgeio/compass/pkg/infra/configdb"
	"github.com/coredgeio/compass/pkg/infra/notifier"
	"github.com/google/uuid"

	"github.com/coredgeio/workflow-manager/pkg/runtime"
)

type BaseImageVersionKey struct {
	Domain  string `bson:"domain,omitempty"`
	Name    string `bson:"name,omitempty"`
	Version string `bson:"version,omitempty"`
}

type BaseImageVersion struct {
	Key         BaseImageVersionKey `bson:"key,omitempty"`
	Id          *uuid.UUID          `bson:"id,omitempty"`
	Desc        *string             `bson:"desc,omitempty"`
	CreateTime  int64               `bson:"createTime,omitempty"`
	CreatedBy   string              `bson:"createdBy,omitempty"`
	ExternalRef string              `bson:"exrernalRef,omitempty"`
}

type BaseImageVersionTable struct {
	notifier.NotifierImpl
	colName string
	dbConn  configdb.DBconnType
}

func (t *BaseImageVersionTable) AllocateKey() interface{} {
	return &BaseImageVersionKey{}
}

// Callback that will be triggered by the watch stream on mongoDB
// meant for internal use only
func (t *BaseImageVersionTable) Callback(op string, key interface{}) {
	nsKey, ok := key.(*BaseImageVersionKey)
	if !ok {
		log.Println("Error: invalid base image version key received from mongodb")
		return
	}

	t.NotifyKey(*nsKey)
}

// implement Get All Keys for Notifier init
func (t *BaseImageVersionTable) NotifierGetAllKeys() []interface{} {
	var list []interface{}
	var entryList []BaseImageVersion
	err := t.dbConn.FindMany(t.colName, bson.D{}, &entryList)
	if err != nil {
		log.Println("Error: unable to find base image version records")
	}
	for _, entry := range entryList {
		list = append(list, entry.Key)
	}
	return list
}

// Add an entry into the table
func (t *BaseImageVersionTable) Add(entry *BaseImageVersion) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	// add current timestamp
	entry.CreateTime = time.Now().Unix()
	// insert new entry
	err := t.dbConn.Insert(t.colName, &entry.Key, entry)
	return err
}

// Update update an entry into the table
func (t *BaseImageVersionTable) Update(entry *BaseImageVersion) error {
	// ensure that we don't update create time during update
	entry.CreateTime = 0
	// trigger update to the entry
	err := t.dbConn.Update(t.colName, &entry.Key, entry)
	return err
}

// Find an entry in table
func (t *BaseImageVersionTable) Find(key *BaseImageVersionKey) (*BaseImageVersion, error) {
	entry := &BaseImageVersion{}
	err := t.dbConn.Find(t.colName, key, entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

// FindById finds an entry of base image version by id
func (t *BaseImageVersionTable) FindById(id *uuid.UUID) (*BaseImageVersion, error) {
	filter := bson.D{
		{
			Key:   "id",
			Value: bson.D{{Key: "$eq", Value: *id}},
		},
	}
	var list []BaseImageVersion
	err := t.dbConn.FindMany(t.colName, filter, &list)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, pkgerrors.Wrap(pkgerrors.NotFound, "entry not found")
	}
	return &list[0], nil
}

// GetListInDomain get all Base image versions in given domain
func (t *BaseImageVersionTable) GetListInDomain(domain string, offset, limit int64) ([]BaseImageVersion, error) {
	filter := bson.D{
		{
			Key:   "key.domain",
			Value: bson.D{{Key: "$eq", Value: domain}},
		},
	}
	dbOptions := &configdb.DbOptions{
		Offset: &offset,
		Limit:  &limit,
	}
	var list []BaseImageVersion
	err := t.dbConn.FindMany(t.colName, filter, &list, dbOptions)
	return list, err
}

// GetList get all Base image version
func (t *BaseImageVersionTable) GetList(offset, limit int64) ([]BaseImageVersion, error) {
	filter := bson.D{}
	dbOptions := &configdb.DbOptions{
		Offset: &offset,
		Limit:  &limit,
	}
	var list []BaseImageVersion
	err := t.dbConn.FindMany(t.colName, filter, &list, dbOptions)
	return list, err
}

// GetCount gets count of all base image versions
func (t *BaseImageVersionTable) GetCount() (int64, error) {
	filter := bson.D{}
	return t.dbConn.Count(t.colName, filter, nil)
}

// GetCountInDomain gets count of base image versions in given domain
func (t *BaseImageVersionTable) GetCountInDomain(domain string) (int64, error) {
	filter := bson.D{
		{
			Key:   "key.domain",
			Value: bson.D{{Key: "$eq", Value: domain}},
		},
	}
	return t.dbConn.Count(t.colName, filter, nil)
}

// Remove an entry from table
func (t *BaseImageVersionTable) Remove(key *BaseImageVersionKey) error {
	err := t.dbConn.Delete(t.colName, key)
	return err
}

// init triggers initialisation for ContainerProject table
func (t *BaseImageVersionTable) init() error {
	cancel, err := t.dbConn.Watch(t.colName, t)
	if err != nil {
		return err
	}
	// load all the existing entries from the store to local map
	var list []BaseImageVersion
	// find all existing entries in this collection
	err = t.dbConn.FindMany(t.colName, bson.D{}, &list)
	if err != nil {
		// this ideally should not have happended
		// cancel watch and return error
		cancel()
		return err
	}
	for i := range list {
		t.NotifyKey(&list[i].Key)
	}
	return nil
}

var baseImageVersionTable *BaseImageVersionTable

func LocateBaseImageVersionTable() (*BaseImageVersionTable, error) {
	if baseImageVersionTable != nil {
		return baseImageVersionTable, nil
	}
	baseImageVersionTable = &BaseImageVersionTable{
		colName: runtime.BaseImageVersionCollection,
		dbConn:  configdb.GetDataStore(runtime.WorkflowEngineDatabaseName),
	}
	baseImageVersionTable.Parent = baseImageVersionTable

	err := baseImageVersionTable.init()
	if err != nil {
		return nil, err
	}
	return baseImageVersionTable, nil
}
