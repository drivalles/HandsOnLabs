package ldspec

import (
	"time"

	"gorm.io/gorm"
)

// DbUser is a GHL user of the tenant location, the caller of the dialer.
type DbUser struct {
	gorm.Model
	// Email is the email of the user.
	Email *string `gorm:"type:varchar(255)"`
	// Role is the GHL role of the user.
	Role *string `gorm:"type:varchar(40)"`
	// Type is the GHL type of the user.
	Type *string `gorm:"type:varchar(40)"`
	// GHLId is the unique GHL user id.
	GHLId *string `gorm:"type:varchar(50);uniqueIndex"`
	// UserName is the display name of the user.
	UserName *string `gorm:"type:varchar(255)"`
	// LastCallId is the id of the latest call of the user.
	LastCallId *uint
	// LastCall is the latest call of the user.
	LastCall     *DbCall
	LDAccessType *LDUserAccess `gorm:"type:varchar(15)"`
}

// TableName is the table of the users.
func (DbUser) TableName() string {
	return "users"
}

// DbContact is a GHL contact of the tenant location, the lead to call.
type DbContact struct {
	gorm.Model
	// GHLId is the unique GHL contact id.
	GHLId string `gorm:"type:varchar(50);uniqueIndex"`
	// FirstName is the first name of the contact.
	FirstName string `gorm:"type:varchar(255)"`
	// LastName is the last name of the contact.
	LastName string `gorm:"type:varchar(255)"`
	// PhoneNumber is the number as GHL sends it, with or without +1.
	PhoneNumber string `gorm:"type:varchar(50)"`
	// DateRegistered is when the contact was added in GHL.
	DateRegistered *time.Time
	// DND tells whether the contact must not be called (do not disturb).
	DND *bool `gorm:"type:boolean"`
	// AssignedTo is the GHL id of the user who owns the contact.
	AssignedTo *string `gorm:"type:varchar(50)"`
	// LevelOfInterest is the interest of the contact, used in the call
	// priority score.
	LevelOfInterest int16 `gorm:"type:int2"`
	// NumberOfCalls is the count of calls with the contact.
	NumberOfCalls int16 `gorm:"type:int2"`
	// InboundOfCalls is the count of inbound calls from the contact.
	InboundOfCalls int16 `gorm:"type:int2"`
	// OutboundOfCalls is the count of outbound calls to the contact.
	OutboundOfCalls int16 `gorm:"type:int2"`
	// LastCallId is the id of the latest call with the contact.
	LastCallId *uint
	// LastCall is the latest call with the contact.
	LastCall *DbCall
	// LastCallTime is when the latest call with the contact happened.
	LastCallTime *time.Time
	// LastCallAttemptTime is when the contact was last locked to be called.
	LastCallAttemptTime *time.Time
	CallbackTime        *time.Time
	CallbackUserId      *uint
	CallbackUser        *DbUser
	// LockId is the node id holding the contact while it is called; nil
	// when it is free.
	LockId *string `gorm:"type:varchar(50)"`
	// LockTs is when the lock was taken.
	LockTs *time.Time
	// LockUserId is the id of the user calling the locked contact.
	LockUserId *uint
	// LockUser is the user calling the locked contact.
	LockUser *DbUser
}

// TableName is the table of the contacts.
func (DbContact) TableName() string {
	return "contacts"
}

// DbContactTag is a GHL tag of a contact.
type DbContactTag struct {
	gorm.Model
	// ContactId is the id of the tagged contact.
	ContactId uint `gorm:"index"`
	// Contact is the tagged contact.
	Contact *DbContact
	// Tags is the tag text.
	Tags string `gorm:"type:varchar(1024)"`
}

// TableName is the table of the contact tags.
func (DbContactTag) TableName() string {
	return "contact_tags"
}

// DbPipeline is a GHL opportunities pipeline.
type DbPipeline struct {
	gorm.Model
	// GHLId is the unique GHL pipeline id.
	GHLId string `gorm:"type:varchar(50);uniqueIndex"`
	// Name is the name of the pipeline.
	Name string `gorm:"type:varchar(255)"`
}

// TableName is the table of the pipelines.
func (DbPipeline) TableName() string {
	return "pipelines"
}

// DbPipelineStage is a stage of a GHL pipeline.
type DbPipelineStage struct {
	gorm.Model
	// PipelineId is the id of the pipeline of the stage.
	PipelineId uint
	// Pipeline is the pipeline of the stage.
	Pipeline *DbPipeline
	// GHLId is the unique GHL stage id.
	GHLId string `gorm:"type:varchar(50);uniqueIndex"`
	// Name is the name of the stage.
	Name string `gorm:"type:varchar(255)"`
}

// TableName is the table of the pipeline stages.
func (DbPipelineStage) TableName() string {
	return "pipeline_stages"
}

// DbOpportunity is a GHL opportunity, a contact placed in a pipeline stage.
type DbOpportunity struct {
	gorm.Model
	// GHLId is the unique GHL opportunity id.
	GHLId string `gorm:"type:varchar(50);uniqueIndex"`
	// ContactId is the id of the contact of the opportunity.
	ContactId uint `gorm:"index"`
	// Contact is the contact of the opportunity.
	Contact *DbContact
	// PipelineId is the id of the pipeline of the opportunity.
	PipelineId uint `gorm:"index"`
	// Pipeline is the pipeline of the opportunity.
	Pipeline *DbPipeline
	// StageId is the id of the current stage of the opportunity.
	StageId uint `gorm:"index"`
	// Stage is the current stage of the opportunity.
	Stage *DbPipelineStage
	// AssignedToId is the id of the user who owns the opportunity.
	AssignedToId *uint `gorm:"index"`
	// AssignedTo is the user who owns the opportunity.
	AssignedTo *DbUser
	// CreatedAt is when the opportunity was created in GHL; it overrides
	// gorm.Model.CreatedAt.
	CreatedAt *time.Time
}

// TableName is the table of the opportunities.
func (DbOpportunity) TableName() string {
	return "opportunities"
}

// DbDid is a phone number of the tenant used as the caller id of the calls.
type DbDid struct {
	gorm.Model
	// DidNumber is the unique validated phone number, stored in the did
	// column.
	DidNumber string `gorm:"type:varchar(15);uniqueIndex;column:did"`
	// Enabled tells whether the number can be used to dial.
	Enabled *bool `gorm:"type:boolean"`
	// NumberOfCalls is the count of calls with the number in the last 10
	// days.
	NumberOfCalls int16 `gorm:"type:int2"`
	// InboundOfCalls is the count of inbound calls to the number.
	InboundOfCalls int16 `gorm:"type:int2"`
	// OutboundOfCalls is the count of outbound calls from the number.
	OutboundOfCalls int16 `gorm:"type:int2"`
	// LockId is the node id holding the number while it dials; nil when it
	// is free.
	LockId *string `gorm:"index"`
	// LockTs is when the lock was taken.
	LockTs *time.Time
	// LockUserId is the id of the user dialing with the locked number.
	LockUserId *uint
	// LockUser is the user dialing with the locked number.
	LockUser *DbUser
	// LastCallId is the id of the latest call with the number.
	LastCallId *uint
	// LastCall is the latest call with the number.
	LastCall *DbCall
	IsSpam   *bool `gorm:"type:boolean"`
}

// TableName is the table of the DIDs.
func (DbDid) TableName() string {
	return "dids"
}

// DbDidHealthReport is the health report of a DID. see https://didchecks.com/en/api-doc
type DbDidHealthReport struct {
	gorm.Model
	DidId           *uint
	Did             *DbDid
	IsSpam          *bool `gorm:"type:boolean"`
	SpamScore       *int16
	SpamType        *string `gorm:"type:varchar(40)"`
	NumberType      *string `gorm:"type:varchar(40)"`
	Carrier         *string `gorm:"type:varchar(40)"`
	LineType        *string `gorm:"type:varchar(40)"`
	ReputationLevel *string `gorm:"type:varchar(40)"`
	NegativeReviews *int16
	PositiveReviews *int16
}

func (DbDidHealthReport) TableName() string {
	return "dids_health_reports"
}

// DbCall is a call, imported from GHL or made by the dialer. The names and
// numbers are copied so the call keeps them when the related records change.
type DbCall struct {
	gorm.Model
	// GHLId is the unique GHL message id of the call; nil until the call is
	// in GHL.
	GHLId *string `gorm:"type:varchar(50);uniqueIndex"`
	// SIPSessionId is the unique SIP session id of a call made by the
	// dialer.
	SIPSessionId *string `gorm:"type:varchar(60);uniqueIndex"`
	// UserId is the id of the user who made or took the call.
	UserId *uint
	// User is the user who made or took the call.
	User *DbUser
	// GHLUserId is the GHL id of the user.
	GHLUserId *string `gorm:"type:varchar(50)"`
	// UserName is the name of the user.
	UserName *string `gorm:"type:varchar(255)"`
	// DidId is the id of the DID of the call.
	DidId *uint
	// Did is the DID of the call.
	Did *DbDid
	// DidNumber is the phone number of the DID.
	DidNumber *string `gorm:"type:varchar(15);index"`
	// ContactId is the id of the contact called.
	ContactId *uint
	// Contact is the contact called.
	Contact *DbContact
	// GHLContactId is the GHL id of the contact.
	GHLContactId *string `gorm:"type:varchar(50);index"`
	// FirstName is the first name of the contact.
	FirstName *string `gorm:"type:varchar(255)"`
	// LastName is the last name of the contact.
	LastName *string `gorm:"type:varchar(255)"`
	// PhoneNumber is the number of the contact.
	PhoneNumber string `gorm:"type:varchar(50);index"`
	// DateTime is when the call started.
	DateTime *time.Time
	// CallPriorityScore is the score the contact had when it was selected to
	// be called.
	CallPriorityScore *int `gorm:"type:int4"`
	// DurationSeconds is the length of the call in seconds.
	DurationSeconds *int64 `gorm:"type:int8"`
	// CallStatus is the final status of the call, as completed.
	CallStatus *string `gorm:"type:varchar(30);index"`
	// MessageType is the GHL message type of the call.
	MessageType *string `gorm:"type:varchar(20)"`
	// Outbound tells whether the call was made to the contact.
	Outbound *bool `gorm:"type:boolean"`
	// Source is where the call came from, as gohighlevel for imported calls.
	Source *string `gorm:"type:varchar(20)"`
	// DispositionName is the name of the disposition given to the call.
	DispositionName *string `gorm:"type:varchar(100)"`
	// DispositionId is the id of the disposition given to the call.
	DispositionId *uint
	// Disposition is the disposition given to the call.
	Disposition *DbDisposition
}

// TableName is the table of the calls.
func (DbCall) TableName() string {
	return "calls"
}

// DbCallHistoryFilter selects the calls of FindCallHistory; the nil and
// empty fields do not filter.
type DbCallHistoryFilter struct {
	// From keeps the calls made at or after it.
	From *time.Time
	// To keeps the calls made before it.
	To *time.Time
	// CallStatus keeps the calls with the status.
	CallStatus *string
	// Outbound keeps the outbound calls when true, the inbound when false.
	Outbound *bool
	// UserGHLIds keeps the calls of the users with the GHL ids.
	UserGHLIds []string
	// ContactGHLIds keeps the calls of the contacts with the GHL ids.
	ContactGHLIds []string
	// Limit is the size of the page and Offset the calls skipped before it.
	Limit  int
	Offset int
}

// DbCacheTime is the expiration of a named cache of GHL data.
type DbCacheTime struct {
	// CacheName is the unique name of the cache.
	CacheName string `gorm:"type:varchar(100);primarykey"`
	// CacheTime is when the cache expires.
	CacheTime time.Time
}

// TableName is the table of the cache times.
func (DbCacheTime) TableName() string {
	return "cache_times"
}

// DbTenantSettings are the settings of the tenant, a single record with the
// PK default.
type DbTenantSettings struct {
	// PK is the key of the settings record, always default.
	PK string `gorm:"primaryKey;size:50"`
	// NewLeadsStageId is the id of the stage where the new leads are placed.
	NewLeadsStageId *uint
	// NewLeadsStage is the stage where the new leads are placed.
	NewLeadsStage *DbPipelineStage
	// DispositionFieldGHLId is the GHL custom field where the call
	// disposition is written.
	DispositionFieldGHLId *string `gorm:"type:varchar(50)"`
}

// TableName is the table of the tenant settings.
func (DbTenantSettings) TableName() string {
	return "tenant_settings"
}

// CallerStatus is the presence of the caller in the dialer.
type CallerStatus string

const (
	// CallerStatusActive is a caller in a calling session.
	CallerStatusActive CallerStatus = "active"
	// CallerStatusAvailable is a caller present without a calling session.
	CallerStatusAvailable CallerStatus = "available"
	// CallerStatusAway is a caller who does not take calls.
	CallerStatusAway CallerStatus = "away"
)

// CallStatus is the state of the current call of a calling session.
type CallStatus string

const (
	// CallStatusPaused is a session waiting for the caller to resume.
	CallStatusPaused CallStatus = "paused"
	// CallStatusDialing is a call being placed.
	CallStatusDialing CallStatus = "dialing"
	// CallStatusRinging is a call ringing at the contact.
	CallStatusRinging CallStatus = "ringing"
	// CallStatusTalking is a call answered by the contact.
	CallStatusTalking CallStatus = "talking"
	// CallStatusTransfer is a call being transferred.
	CallStatusTransfer CallStatus = "transfer"
	// CallStatusHangup is a call that ended.
	CallStatusHangup CallStatus = "hangup"
)

// DbDialerSession is the dialer session of a user in a node, with the calling
// params and the current status.
type DbDialerSession struct {
	gorm.Model
	// CompanyId is the GHL company (agency) id.
	CompanyId string `gorm:"type:varchar(40);index"`
	// LocationId is the GHL location id.
	LocationId string `gorm:"type:varchar(40);index"`
	// NodeId is the id of the server node running the session.
	NodeId string `gorm:"type:varchar(40);index"`
	// SessionId is the unique id of the session.
	SessionId string `gorm:"type:varchar(50);uniqueIndex"`
	// UserId is the id of the user of the session.
	UserId *uint
	// User is the user of the session.
	User *DbUser
	// PipelineId is the id of the pipeline selected to call.
	PipelineId *uint
	// Pipeline is the pipeline selected to call.
	Pipeline *DbPipeline
	// StageId is the id of the stage selected to call.
	StageId *uint
	// stage is the stage selected to call; it is unexported, so gorm does
	// not load it.
	Stage *DbPipelineStage
	// Tags are the contact tags selected to call.
	Tags string `gorm:"type:varchar(1024)"`
	// AssignedToMe tells whether only the opportunities of the user are
	// called.
	AssignedToMe bool
	// NewOpportunities tells whether only the new leads are called.
	NewOpportunities bool
	// CallerStatus is the presence of the caller; nil when the session is
	// closed.
	CallerStatus *CallerStatus `gorm:"type:varchar(12);index"`
	// CallStatus is the state of the current call.
	CallStatus *CallStatus `gorm:"type:varchar(15);index"`
	// CallResult is the result of the latest call.
	CallResult *string `gorm:"type:varchar(15);index"`
	// LastCallsSIPSessionId is the SIP session id of the latest call, used
	// to find it before LastCallId is set.
	LastCallsSIPSessionId *string `gorm:"type:varchar(60)"`
	// LastCallId is the id of the latest call of the session.
	LastCallId *uint
	// LastCall is the latest call of the session.
	LastCall *DbCall
}

// TableName is the table of the dialer sessions.
func (DbDialerSession) TableName() string {
	return "dialer_sessions"
}

// DbTransferContact is a destination where the calls can be transferred.
type DbTransferContact struct {
	gorm.Model
	// Name is the display name of the destination.
	Name string `gorm:"type:varchar(255)"`
	// PhoneNumber is the number calls are transferred to.
	PhoneNumber *string `gorm:"type:varchar(40)"`
}

// TableName is the table of the transfer contacts.
func (DbTransferContact) TableName() string {
	return "transfer_contacts"
}

// DbDisposition is an outcome the caller can give to a call.
type DbDisposition struct {
	gorm.Model
	// Name is the name of the disposition.
	Name string `gorm:"type:varchar(100)"`
	// GHLId is the unique GHL id of the disposition option.
	GHLId *string `gorm:"type:varchar(50);uniqueIndex"`
}

// TableName is the table of the dispositions.
func (DbDisposition) TableName() string {
	return "dispositions"
}

// DbCallScript is a script the caller reads during the calls.
type DbCallScript struct {
	gorm.Model
	// Name is the name of the script.
	Name string `gorm:"type:varchar(255)"`
	// Text is the content of the script.
	Text string `gorm:"type:text"`
}

// TableName is the table of the call scripts.
func (DbCallScript) TableName() string {
	return "call_scripts"
}

type TenantsManager interface {
	FindOrCreateTenant(locationId string) (TenantRepo, error)
	GetTenant(locationId string) (TenantRepo, error)
}

// TenantsRepo is the repository of a tenant database. Find methods returning
// one record return gorm.ErrRecordNotFound when there is none.
type TenantRepo interface {
	// CacheExpired tells whether the cache is missing or expired; when true
	// it also renews the cache for the duration, so only one caller
	// refreshes it.
	CacheExpired(cacheName string, duration time.Duration) bool
	// UpdateCache sets the cache to expire after the duration.
	UpdateCache(cacheName string, duration time.Duration)
	// DeleteCache removes the cache, so it is expired.
	DeleteCache(cacheName string)
	// FindNextExpirationTime returns the next expiration time for the specified cache, or an error if retrieval fails.
	FindNextExpirationTime(cacheName string) (time.Time, error)

	// SaveCall creates or updates the call.
	SaveCall(record *DbCall) error
	// DeleteCall permanently deletes the call.
	DeleteCall(record *DbCall) error
	// DeleteCallById permanently deletes the call by its id.
	DeleteCallById(id uint) error
	// CountCalls returns the number of calls.
	CountCalls() (int64, error)
	// FindCalls returns the calls with the GHL ids, newest first.
	FindCalls(ghlCallIds []string) ([]*DbCall, error)
	// FindCall finds the call by its GHL id.
	FindCall(ghlCallId string) (*DbCall, error)
	// FindCallById finds the call by its id.
	FindCallById(id uint) (*DbCall, error)
	// FindCallsByIds returns the calls with the ids, newest first.
	FindCallsByIds(ids []uint) ([]*DbCall, error)
	// FindCallBySIPSessionId finds the call by its SIP session id.
	FindCallBySIPSessionId(sipSessionId string) (*DbCall, error)
	// FindLatestCallHistory returns up to limit calls of the user, newest
	// first.
	FindLatestCallHistory(userId uint, limit int) ([]*DbCall, error)
	// FindCallsByContact returns the calls of the contact with the GHL id,
	// newest first.
	FindCallsByContact(ghlContactId string) ([]*DbCall, error)
	// FindCallHistory returns the page of the calls matching the filter,
	// newest first, with the number of calls matching it across all pages.
	FindCallHistory(filter *DbCallHistoryFilter) ([]*DbCall, int64, error)
	// FindCallsByDate returns the calls with a contact in the 24 hours from
	// the date, newest first.
	FindCallsByDate(date time.Time) ([]*DbCall, error)

	// UpdateContactCallStats recounts the calls of the contact and sets its
	// last call; a nil or unknown id is ignored.
	UpdateContactCallStats(ghlContactId *string)
	// FindContactByGHLId finds the contact by its GHL id.
	FindContactByGHLId(ghlContactId string) (*DbContact, error)
	// FindContactById finds the contact by its id.
	FindContactById(id uint) (*DbContact, error)
	// FindContactByPhoneNumber finds the newest contact with the number in
	// any of the US formats.
	FindContactByPhoneNumber(phoneNumber string) (*DbContact, error)
	// FindContactTags returns the tags of the contact with the id; nil on
	// error.
	FindContactTags(ghlContactId uint) []string
	// UpdateContactsNoCalls recounts the calls of every contact.
	UpdateContactsNoCalls()
	// LockContact locks the contact for the lock id (the node id) and user;
	// it returns false when another lock holds it.
	LockContact(ghlContactId string, lockId string, userId uint) (bool, error)
	// UnlockContact releases the contact; it returns false when the lock id
	// does not hold it.
	UnlockContact(ghlContactId string, lockId string) (bool, error)
	// ClearContactCallback clears the callback of the contact, if it has one.
	ClearContactCallback(ghlContactId string) error
	// UpdateContactCallback schedules the callback of the contact for the
	// user; it returns gorm.ErrRecordNotFound when the contact does not exist.
	UpdateContactCallback(ghlContactId string, userId uint, callbackTime time.Time) error
	// SaveContact creates or updates the contact.
	SaveContact(contact *DbContact) error

	// UpdateDidCallStats recounts the calls of the DID in the last 10 days
	// and sets its last call.
	UpdateDidCallStats(phoneNumber string)
	// FindOrCreateDid finds the DID of the number, creating it with enabled
	// when missing; the number must be valid.
	FindOrCreateDid(from *string, enabled bool) (*DbDid, error)
	// FindEnabledDids returns the DIDs that can be used to dial.
	FindEnabledDids() ([]*DbDid, error)
	// FindAllDids returns all the DIDs.
	FindAllDids() ([]*DbDid, error)
	// SaveDid creates or updates the DID.
	SaveDid(did *DbDid) error
	// FindDid finds the DID by its number.
	FindDid(number string) (*DbDid, error)
	// LockDid locks the DID for the lock id (the node id) and user, first
	// releasing a lock older than 10 minutes; it returns false when another
	// lock holds it.
	LockDid(number string, lockId string, userId uint) (bool, error)
	// UnlockDid releases the DID; it returns false when the lock id does not
	// hold it.
	UnlockDid(number string, lockId string) (bool, error)
	FindDidHealthReports(didId uint) ([]*DbDidHealthReport, error)
	SaveDidHealthReport(report *DbDidHealthReport) error

	// LockMaintenance releases the contacts and DIDs locked for more than 2
	// hours whose last call is completed.
	LockMaintenance(nodeId string)
	// ReleaseAllLocks releases the contacts and DIDs locked by the node.
	ReleaseAllLocks(nodeId string)

	// DeleteOpportunity permanently deletes the opportunity.
	DeleteOpportunity(record *DbOpportunity) error
	// SaveOpportunity creates or updates the opportunity.
	SaveOpportunity(record *DbOpportunity) error
	// FindOpportunitiesByPipeline returns the opportunities of the pipeline.
	FindOpportunitiesByPipeline(pipeline *DbPipeline) ([]*DbOpportunity, error)
	// FindOpportunityByGHLId finds the opportunity by its GHL id.
	FindOpportunityByGHLId(ghlOpportunityId string) (*DbOpportunity, error)
	// FindOpportunity finds the opportunity of the contact in the pipeline.
	FindOpportunity(pipeline *DbPipeline, contact *DbContact) (*DbOpportunity, error)
	// FindOpportunities returns the opportunities in the stage of the
	// pipeline, only those of assignedTo when it is not nil.
	FindOpportunities(pipeline *DbPipeline, stage *DbPipelineStage, assignedTo *DbUser) ([]*DbOpportunity, error)
	// FindOpportunitiesByContact returns the opportunities of the contact.
	FindOpportunitiesByContact(contact *DbContact) ([]*DbOpportunity, error)
	// FindNewOpportunities returns the opportunities in the new leads stage
	// whose contact registered in the last 24 hours; empty when the stage is
	// not set.
	FindNewOpportunities() ([]*DbOpportunity, error)
	// FindCallbackOpportunities returns the opportunities of the contacts
	// whose callback is due, only the callbacks of callbackUser when it is
	// not nil.
	FindCallbackOpportunities(callbackUser *DbUser) ([]*DbOpportunity, error)

	// DeletePipeline permanently deletes the pipeline.
	DeletePipeline(record *DbPipeline) error
	// SavePipeline creates or updates the pipeline.
	SavePipeline(record *DbPipeline) error
	// FindPipelines returns all the pipelines.
	FindPipelines() ([]*DbPipeline, error)
	// FindPipelineById finds the pipeline by its id.
	FindPipelineById(id uint) (*DbPipeline, error)
	// FindPipeline finds the pipeline by its GHL id.
	FindPipeline(ghlPipelineId string) (*DbPipeline, error)

	// DeleteStage permanently deletes the stage.
	DeleteStage(record *DbPipelineStage) error
	// SaveStage creates or updates the stage.
	SaveStage(record *DbPipelineStage) error
	// FindStagesByPipelineId returns the stages of the pipeline with the id.
	FindStagesByPipelineId(pipelineId uint) ([]*DbPipelineStage, error)
	// DeleteStagesByPipelineId permanently deletes the stages of the
	// pipeline with the id.
	DeleteStagesByPipelineId(id uint) error
	// FindStageByGHLId finds the stage of the pipeline by its GHL id.
	FindStageByGHLId(pipelineId uint, ghlStageId string) (*DbPipelineStage, error)
	// FindStageById finds the stage by its id, with its pipeline.
	FindStageById(stageId uint) (*DbPipelineStage, error)
	// FindStagesByPipeline returns the stages of the pipeline; nil on error.
	FindStagesByPipeline(record *DbPipeline) []*DbPipelineStage

	// FindUser finds the user by its GHL id.
	FindUser(ghlUserId *string) (*DbUser, error)
	// FindOrCreateUser finds the user by its GHL id, creating it when
	// missing; the name is set when the user has none.
	FindOrCreateUser(ghlUserId *string, name string) (*DbUser, error)
	// SaveUser creates or updates the user.
	SaveUser(user *DbUser) error
	// FindUsers returns all the users.
	FindUsers() ([]*DbUser, error)
	// UpdateUserAccess sets the access of the user with the id; it returns
	// gorm.ErrRecordNotFound when the user does not exist.
	UpdateUserAccess(id uint, access LDUserAccess) error

	// GetNewLeadsStage returns the new leads stage with its pipeline; nil
	// when it is not set.
	GetNewLeadsStage() (*DbPipelineStage, error)
	// SetNewLeadsStage sets the new leads stage.
	SetNewLeadsStage(stageId uint) error
	// GetDispositionField returns the GHL id of the disposition custom
	// field; empty when it is not set.
	GetDispositionField() (string, error)
	// SetDispositionField sets the GHL id of the disposition custom field.
	SetDispositionField(ghlId string) error
	// CountAbandonedLeads returns the number of contacts registered more
	// than an hour ago and never called.
	CountAbandonedLeads() (int64, error)

	// FindDialerSessionBySessionId finds the dialer session by its session
	// id.
	FindDialerSessionBySessionId(sessionId string) (*DbDialerSession, error)
	// FindDialerSessions returns the dialer sessions of the user, latest
	// updated first.
	FindDialerSessions(userId *uint) ([]*DbDialerSession, error)
	// FindOpenDialerSessions returns the dialer sessions of the location
	// with a caller status, latest updated first.
	FindOpenDialerSessions(locationId string) ([]*DbDialerSession, error)
	// SaveDialerSession creates or updates the dialer session.
	SaveDialerSession(session *DbDialerSession) error
	// RemoveOldDialerSessions permanently deletes the dialer sessions of the
	// node and those without a node, as when the node starts.
	RemoveOldDialerSessions(nodeId string) error

	// FindTransferContacts returns all the transfer contacts.
	FindTransferContacts() ([]*DbTransferContact, error)
	// SaveTransferContact creates or updates the transfer contact.
	SaveTransferContact(contact *DbTransferContact) error

	// FindDispositions returns all the dispositions.
	FindDispositions() ([]*DbDisposition, error)
	// FindDispositionById finds the disposition by its id.
	FindDispositionById(id uint) (*DbDisposition, error)
	// SaveDisposition creates or updates the disposition.
	SaveDisposition(disposition *DbDisposition) error
	// FindOrCreateDispositionByName finds the disposition with the name,
	// creating it when it does not exist.
	FindOrCreateDispositionByName(name string) (*DbDisposition, error)

	// FindCallScripts returns all the call scripts.
	FindCallScripts() ([]*DbCallScript, error)
	// SaveCallScript creates or updates the call script.
	SaveCallScript(script *DbCallScript) error
}

var DbTenantTables = []interface{}{
	DbContactTag{},
	DbDid{},
	DbOpportunity{},
	DbPipelineStage{},
	DbCall{},
	DbPipeline{},
	DbContact{},
	DbUser{},
	DbCacheTime{},
	DbTenantSettings{},
	DbDialerSession{},
	DbCallScript{},
	DbTransferContact{},
	DbDisposition{},
	DbDidHealthReport{},
}
