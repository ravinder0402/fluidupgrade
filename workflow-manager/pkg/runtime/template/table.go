package template

import (
	"log"
	"time"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson"

	pkgerrors "github.com/coredgeio/compass/pkg/errors"
	configdb "github.com/coredgeio/compass/pkg/infra/configdb"
	"github.com/coredgeio/compass/pkg/infra/notifier"

	"github.com/coredgeio/workflow-manager/pkg/runtime"
)

type TemplateKey struct {
	Domain  string `bson:"domain,omitempty"`
	Project string `bson:"project,omitempty"`
	Name    string `bson:"name,omitempty"`
}

type TemplateNodeType int32

const (
	ModuleNode TemplateNodeType = iota
	CatalogNode

	UserInputNode TemplateNodeType = 100
)

type TemplateUserInputNodeData struct {
	Name       string `bson:"name,omitempty"`
	Desc       string `bson:"desc,omitempty"`
	DefaultVal string `bson:"defaultVal,omitempty"`
	Opt        bool   `bson:"opt,omitempty"`
}

type TemplateNode struct {
	Type     TemplateNodeType           `bson:"type,omitempty"`
	Name     string                     `bson:"name,omitempty"`
	NodeId   string                     `bson:"nodeId,omitempty"`
	ModuleId string                     `bson:"moduleId,omitempty"`
	Stage    int32                      `bson:"stage,omitempty"`
	X        float64                    `bson:"x,omitempty"`
	Y        float64                    `bson:"y,omitempty"`
	UserData *TemplateUserInputNodeData `bson:"userData,omitempty"`
}

type TemplateLink struct {
	Source    string `bson:"source,omitempty"`
	SourceVar string `bson:"sourceVar,omitempty"`
	Target    string `bson:"target,omitempty"`
	TargetVar string `bson:"targetVar,omitempty"`
}

type TemplateEntry struct {
	Key        TemplateKey     `bson:"key,omitempty"`
	Id         *uuid.UUID      `bson:"id,omitempty"`
	Desc       *string         `bson:"desc,omitempty"`
	CreatedBy  string          `bson:"createdBy,omitempty"`
	CreateTime int64           `bson:"createTime,omitempty"`
	LastUpdate int64           `bson:"lastUpdate,omitempty"`
	Tags       []string        `bson:"tags,omitempty"`
	IsDeleted  bool            `bson:"isDeleted,omitempty"`
	Nodes      []*TemplateNode `bson:"nodes,omitempty"`
	Links      []*TemplateLink `bson:"link,omitempty"`
}

type TemplateTable struct {
	notifier.NotifierImpl
	colName string
	dbConn  configdb.DBconnType
}

func (t *TemplateTable) AllocateKey() interface{} {
	return &TemplateKey{}
}

func (t *TemplateTable) Callback(op string, key interface{}) {
	modKey, ok := key.(*TemplateKey)
	if !ok {
		log.Println("Error: invalid template key received from mongodb")
		return
	}

	t.NotifyKey(*modKey)
}

func (t *TemplateTable) NotifierGetAllKeys() []interface{} {
	var list []interface{}
	var entryList []TemplateEntry
	err := t.dbConn.FindMany(t.colName, bson.D{}, &entryList)
	if err != nil {
		log.Println("Error: unable to find template records")
	}
	for _, entry := range entryList {
		list = append(list, entry.Key)
	}
	return list
}

func (t *TemplateTable) Add(entry *TemplateEntry) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	entry.CreateTime = time.Now().Unix()
	entry.LastUpdate = time.Now().Unix()
	err := t.dbConn.Insert(t.colName, entry.Key, entry)
	return err
}

func (t *TemplateTable) Update(entry *TemplateEntry) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	entry.CreateTime = 0
	entry.LastUpdate = time.Now().Unix()
	err := t.dbConn.Update(t.colName, &entry.Key, entry)
	return err
}

func (t *TemplateTable) Find(key *TemplateKey) (*TemplateEntry, error) {
	entry := &TemplateEntry{}
	err := t.dbConn.Find(t.colName, key, entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (t *TemplateTable) FindById(id *uuid.UUID) (*TemplateEntry, error) {
	filter := bson.D{
		{
			Key:   "id",
			Value: bson.D{{Key: "$eq", Value: *id}},
		},
	}
	var list []TemplateEntry
	err := t.dbConn.FindMany(t.colName, filter, &list)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, pkgerrors.Wrap(pkgerrors.NotFound, "No records found")
	}
	return &list[0], nil
}

func (t *TemplateTable) GetListInProject(domain, project string, offset, limit int64) ([]TemplateEntry, error) {
	filter := bson.D{
		{
			Key:   "key.domain",
			Value: bson.D{{Key: "$eq", Value: domain}},
		},
		{
			Key:   "key.project",
			Value: bson.D{{Key: "$eq", Value: project}},
		},
	}
	dbOptions := &configdb.DbOptions{
		Offset: &offset,
		Limit:  &limit,
	}
	var list []TemplateEntry
	err := t.dbConn.FindMany(t.colName, filter, &list, dbOptions)
	return list, err
}

func (t *TemplateTable) GetList(offset, limit int64) ([]TemplateEntry, error) {
	filter := bson.D{}
	dbOptions := &configdb.DbOptions{
		Offset: &offset,
		Limit:  &limit,
	}
	var list []TemplateEntry
	err := t.dbConn.FindMany(t.colName, filter, &list, dbOptions)
	return list, err
}

func (t *TemplateTable) GetCount() (int64, error) {
	return t.dbConn.Count(t.colName, bson.D{}, nil)
}

func (t *TemplateTable) GetCountInProject(domain, project string) (int64, error) {
	filter := bson.D{
		{
			Key:   "key.domain",
			Value: bson.D{{Key: "$eq", Value: domain}},
		},
		{
			Key:   "key.project",
			Value: bson.D{{Key: "$eq", Value: project}},
		},
	}
	return t.dbConn.Count(t.colName, filter, nil)
}

func (t *TemplateTable) Remove(key *TemplateKey) error {
	return t.dbConn.Delete(t.colName, key)
}

func (t *TemplateTable) init() error {
	cancel, err := t.dbConn.Watch(t.colName, t)
	if err != nil {
		return err
	}
	// load all the existing entries from the store to local map
	var list []TemplateEntry
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

var templateTable *TemplateTable

func LocateTemplateTable() (*TemplateTable, error) {
	if templateTable != nil {
		return templateTable, nil
	}
	templateTable = &TemplateTable{
		colName: runtime.DummyTemplateCollection,
		dbConn:  configdb.GetDataStore(runtime.WorkflowEngineDatabaseName),
	}
	templateTable.Parent = templateTable

	err := templateTable.init()
	if err != nil {
		return nil, err
	}
	return templateTable, nil
}
