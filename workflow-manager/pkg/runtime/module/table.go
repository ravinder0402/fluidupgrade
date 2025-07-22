package module

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

type ModuleKey struct {
	Domain  string `bson:"domain,omitempty"`
	Project string `bson:"project,omitempty"`
	Name    string `bson:"name,omitempty"`
}

type InputKeyData struct {
	DataType   string `bson:"datatype,omitempty"`
	Opt        bool   `bson:"opt,omitempty"`
	DefaultVal string `bson:"defaultVal,omitempty"`
}
type InputKeysType map[string]*InputKeyData

type OutputKeyData struct {
	DataType  string `bson:"datatype,omitempty"`
	ValueFrom string `bson:"defaultVal,omitempty"`
}
type OutputKeysType map[string]*OutputKeyData

type ModuleGitInfo struct {
	Url        string `bson:"url"`
	GitRef     string `bson:"gitref"`
	WorkingDir string `bson:"workdir"`
}

type ModuleFileInfo struct {
	Name    string `bson:"name"`
	Content []byte `bson:"content"`
	Perm    string `bson:"perm"`
}

type ModuleBuildConfig struct {
	BaseImage   string            `bson:"baseimage,omitempty"`
	BuildScript []string          `bson:"buildscript,omitempty"`
	Env         map[string]string `bson:"env,omitempty"`
	EntryPoint  []string          `bson:"entrypoint,omitempty"`
	GitInfo     *ModuleGitInfo    `bson:"gitInfo,omitempty"`
	Files       []*ModuleFileInfo `bson:"files,omitempty"`
}

type ModuleBuildStatusType int32

const (
	ModuleBuildPending = iota
	ModuleBuildInProgress
	ModuleBuildCompleted
	ModuleBuildFailed
)

type ModuleBuildStatus struct {
	Status    ModuleBuildStatusType `bson:"status,omitempty"`
	Logs      string                `bson:"logs,omitempty"`
	BuildTime int64                 `bson:"buildTime,omitempty"`
	Config    *ModuleBuildConfig    `bson:"config,omitempty"`
}

type ModuleEntry struct {
	Key         ModuleKey          `bson:"key,omitempty"`
	Id          *uuid.UUID         `bson:"id,omitempty"`
	Desc        *string            `bson:"desc,omitempty"`
	CreatedBy   string             `bson:"createdBy,omitempty"`
	CreateTime  int64              `bson:"createTime,omitempty"`
	LastUpdate  int64              `bson:"lastUpdate,omitempty"`
	Tags        []string           `bson:"tags,omitempty"`
	InputKeys   InputKeysType      `bson:"inputKeys,omitempty"`
	OutputKeys  OutputKeysType     `bson:"outputKeys,omitempty"`
	BuildConfig *ModuleBuildConfig `bson:"buildConfig,omitempty"`
	BuildStatus *ModuleBuildStatus `bson:"buildStatus,omitempty"`
	IsDeleted   bool               `bson:"isDeleted,omitempty"`
}

type moduleBuildStatusReset struct {
	Key         ModuleKey          `bson:"key,omitempty"`
	BuildStatus *ModuleBuildStatus `bson:"buildStatus"`
}

type moduleInputKeys struct {
	Key       ModuleKey     `bson:"key,omitempty"`
	InputKeys InputKeysType `bson:"inputKeys"`
}

type moduleOutputKeys struct {
	Key        ModuleKey      `bson:"key,omitempty"`
	OutputKeys OutputKeysType `bson:"outputKeys"`
}

type ModuleTable struct {
	notifier.NotifierImpl
	colName string
	dbConn  configdb.DBconnType
}

func (m *ModuleTable) AllocateKey() interface{} {
	return &ModuleKey{}
}

func (m *ModuleTable) Callback(op string, key interface{}) {
	modKey, ok := key.(*ModuleKey)
	if !ok {
		log.Println("Error: invalid module key received from mongodb")
		return
	}

	m.NotifyKey(*modKey)
}

func (m *ModuleTable) NotifierGetAllKeys() []interface{} {
	var list []interface{}
	var entryList []ModuleEntry
	err := m.dbConn.FindMany(m.colName, bson.D{}, &entryList)
	if err != nil {
		log.Println("Error: unable to find module records")
	}
	for _, entry := range entryList {
		list = append(list, entry.Key)
	}
	return list
}

func (m *ModuleTable) Add(entry *ModuleEntry) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	entry.CreateTime = time.Now().Unix()
	entry.LastUpdate = time.Now().Unix()
	err := m.dbConn.Insert(m.colName, entry.Key, entry)
	return err
}

func (m *ModuleTable) Update(entry *ModuleEntry) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	entry.CreateTime = 0
	entry.LastUpdate = time.Now().Unix()
	err := m.dbConn.Update(m.colName, &entry.Key, entry)
	return err
}

func (m *ModuleTable) ResetBuildStatus(key *ModuleKey) error {
	entry := &moduleBuildStatusReset{
		Key: *key,
	}
	err := m.dbConn.Update(m.colName, &entry.Key, entry)
	return err
}

func (m *ModuleTable) EmptyInputKeys(key *ModuleKey) error {
	entry := &moduleInputKeys{
		Key: *key,
	}
	err := m.dbConn.Update(m.colName, &entry.Key, entry)
	return err
}

func (m *ModuleTable) EmptyOutputKeys(key *ModuleKey) error {
	entry := &moduleOutputKeys{
		Key: *key,
	}
	err := m.dbConn.Update(m.colName, &entry.Key, entry)
	return err
}

func (m *ModuleTable) Find(key *ModuleKey) (*ModuleEntry, error) {
	entry := &ModuleEntry{}
	err := m.dbConn.Find(m.colName, key, entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (m *ModuleTable) FindById(id *uuid.UUID) (*ModuleEntry, error) {
	filter := bson.D{
		{
			Key:   "id",
			Value: bson.D{{Key: "$eq", Value: *id}},
		},
	}
	var list []ModuleEntry
	err := m.dbConn.FindMany(m.colName, filter, &list)
	if err != nil {
		return nil, err
	}
	if len(list) == 0 {
		return nil, pkgerrors.Wrap(pkgerrors.NotFound, "No records found")
	}
	return &list[0], nil
}

func (m *ModuleTable) GetListInProject(domain, project string, offset, limit int64) ([]ModuleEntry, error) {
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
	var list []ModuleEntry
	err := m.dbConn.FindMany(m.colName, filter, &list, dbOptions)
	return list, err
}

func (m *ModuleTable) GetList(offset, limit int64) ([]ModuleEntry, error) {
	filter := bson.D{}
	dbOptions := &configdb.DbOptions{
		Offset: &offset,
		Limit:  &limit,
	}
	var list []ModuleEntry
	err := m.dbConn.FindMany(m.colName, filter, &list, dbOptions)
	return list, err
}

func (m *ModuleTable) GetCount() (int64, error) {
	return m.dbConn.Count(m.colName, bson.D{}, nil)
}

func (m *ModuleTable) GetCountInProject(domain, project string) (int64, error) {
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
	return m.dbConn.Count(m.colName, filter, nil)
}

func (m *ModuleTable) Remove(key *ModuleKey) error {
	return m.dbConn.Delete(m.colName, key)
}

func (m *ModuleTable) init() error {
	cancel, err := m.dbConn.Watch(m.colName, m)
	if err != nil {
		return err
	}
	// load all the existing entries from the store to local map
	var list []ModuleEntry
	// find all existing entries in this collection
	err = m.dbConn.FindMany(m.colName, bson.D{}, &list)
	if err != nil {
		// this ideally should not have happended
		// cancel watch and return error
		cancel()
		return err
	}
	for i := range list {
		m.NotifyKey(&list[i].Key)
	}
	return nil
}

var moduleTable *ModuleTable

func LocateModuleTable() (*ModuleTable, error) {
	if moduleTable != nil {
		return moduleTable, nil
	}
	moduleTable = &ModuleTable{
		colName: runtime.ModulesCollection,
		dbConn:  configdb.GetDataStore(runtime.WorkflowEngineDatabaseName),
	}
	moduleTable.Parent = moduleTable

	err := moduleTable.init()
	if err != nil {
		return nil, err
	}
	return moduleTable, nil
}
