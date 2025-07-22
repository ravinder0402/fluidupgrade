package workflow

import (
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson"

	pkgerrors "github.com/coredgeio/compass/pkg/errors"
	configdb "github.com/coredgeio/compass/pkg/infra/configdb"
	"github.com/coredgeio/compass/pkg/infra/notifier"

	"github.com/coredgeio/workflow-manager/pkg/runtime"
)

type WorkflowKey struct {
	Domain  string `bson:"domain,omitempty"`
	Project string `bson:"project,omitempty"`
	Name    string `bson:"name,omitempty"`
}

type WorkflowState int32

const (
	WorkflowCreated WorkflowState = iota
	WorkflowScheduled
	WorkflowRunning
	WorkflowCompleted
	WorkflowFailed
)

type WorkflowNodeType int32

const (
	ModuleNode WorkflowNodeType = iota
	CatalogNode

	UserInputNode WorkflowNodeType = 100
)

type WorkflowValue struct {
	Name  string `bson:"name,omitempty"`
	Value string `bson:"value,omitempty"`
}

type WorkflowNode struct {
	Type     WorkflowNodeType `bson:"type,omitempty"`
	Name     string           `bson:"name,omitempty"`
	NodeId   string           `bson:"nodeId,omitempty"`
	ModuleId string           `bson:"moduleId,omitempty"`
	Stage    int32            `bson:"stage,omitempty"`
	X        float64          `bson:"x,omitempty"`
	Y        float64          `bson:"y,omitempty"`
	State    WorkflowState    `bson:"state,omitempty"`
	Error    string           `bson:"error,omitempty"`
	ArgoId   string           `bson:"argoId,omitempty"`
	Inputs   []*WorkflowValue `bson:"inputs,omitempty"`
	Outputs  []*WorkflowValue `bson:"outputs,omitempty"`
	ExitCode int32            `bson:"exitCode,omitempty"`
}

type WorkflowLink struct {
	Source    string `bson:"source,omitempty"`
	SourceVar string `bson:"sourceVar,omitempty"`
	Target    string `bson:"target,omitempty"`
	TargetVar string `bson:"targetVar,omitempty"`
}

type WorkflowStatus struct {
	State WorkflowState   `bson:"state,omitempty"`
	Nodes []*WorkflowNode `bson:"nodes,omitempty"`
	Links []*WorkflowLink `bson:"link,omitempty"`
}

type WorkflowEntry struct {
	Key        WorkflowKey       `bson:"key,omitempty"`
	Desc       *string           `bson:"desc,omitempty"`
	Template   string            `bson:"template,omitempty"`
	Inputs     map[string]string `bson:"inputs,omitempty"`
	State      WorkflowState     `bson:"state,omitempty"`
	CreatedBy  string            `bson:"createdBy,omitempty"`
	CreateTime int64             `bson:"createTime,omitempty"`
	StartTime  int64             `bson:"startTime,omitempty"`
	EndTime    int64             `bson:"endTime,omitempty"`
	Tags       []string          `bson:"tags,omitempty"`
	IsDeleted  bool              `bson:"isDeleted,omitempty"`
	Status     *WorkflowStatus   `bson:"status,omitempty"`
}

type WorkflowTable struct {
	notifier.NotifierImpl
	colName string
	dbConn  configdb.DBconnType
}

func (t *WorkflowTable) AllocateKey() interface{} {
	return &WorkflowKey{}
}

func (t *WorkflowTable) Callback(op string, key interface{}) {
	modKey, ok := key.(*WorkflowKey)
	if !ok {
		log.Println("Error: invalid workflow key received from mongodb")
		return
	}

	t.NotifyKey(*modKey)
}

func (t *WorkflowTable) NotifierGetAllKeys() []interface{} {
	var list []interface{}
	var entryList []WorkflowEntry
	err := t.dbConn.FindMany(t.colName, bson.D{}, &entryList)
	if err != nil {
		log.Println("Error: unable to find workflow records")
	}
	for _, entry := range entryList {
		list = append(list, entry.Key)
	}
	return list
}

func (t *WorkflowTable) Add(entry *WorkflowEntry) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	entry.CreateTime = time.Now().Unix()
	err := t.dbConn.Insert(t.colName, entry.Key, entry)
	return err
}

func (t *WorkflowTable) Update(entry *WorkflowEntry) error {
	if entry == nil {
		return pkgerrors.Wrap(pkgerrors.InvalidArgument, "Invalid Entry")
	}
	entry.CreateTime = 0
	err := t.dbConn.Update(t.colName, &entry.Key, entry)
	return err
}

func (t *WorkflowTable) Find(key *WorkflowKey) (*WorkflowEntry, error) {
	entry := &WorkflowEntry{}
	err := t.dbConn.Find(t.colName, key, entry)
	if err != nil {
		return nil, err
	}
	return entry, nil
}

func (t *WorkflowTable) GetListInProject(domain, project string, offset, limit int64) ([]WorkflowEntry, error) {
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
	var list []WorkflowEntry
	err := t.dbConn.FindMany(t.colName, filter, &list, dbOptions)
	return list, err
}

func (t *WorkflowTable) GetList(offset, limit int64) ([]WorkflowEntry, error) {
	filter := bson.D{}
	dbOptions := &configdb.DbOptions{
		Offset: &offset,
		Limit:  &limit,
	}
	var list []WorkflowEntry
	err := t.dbConn.FindMany(t.colName, filter, &list, dbOptions)
	return list, err
}

func (t *WorkflowTable) GetCount() (int64, error) {
	return t.dbConn.Count(t.colName, bson.D{}, nil)
}

func (t *WorkflowTable) GetCountInProject(domain, project string) (int64, error) {
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

func (t *WorkflowTable) Remove(key *WorkflowKey) error {
	return t.dbConn.Delete(t.colName, key)
}

func (t *WorkflowTable) init() error {
	cancel, err := t.dbConn.Watch(t.colName, t)
	if err != nil {
		return err
	}
	// load all the existing entries from the store to local map
	var list []WorkflowEntry
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

var workflowTable *WorkflowTable

func LocateWorkflowTable() (*WorkflowTable, error) {
	if workflowTable != nil {
		return workflowTable, nil
	}
	workflowTable = &WorkflowTable{
		colName: runtime.DummyWorkflowCollection,
		dbConn:  configdb.GetDataStore(runtime.WorkflowEngineDatabaseName),
	}
	workflowTable.Parent = workflowTable

	err := workflowTable.init()
	if err != nil {
		return nil, err
	}
	return workflowTable, nil
}
