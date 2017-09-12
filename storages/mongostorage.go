package storages

import (
	"fmt"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"time"
	"os"
	"io"
	"github.com/michivip/sharexhandler"
)

type MongoStorageConfiguration struct {
	// Database name where the upload collection is created
	DatabaseName string
	// Name of the upload collection
	UploadCollectionName string
	// The folder path has to end with a slash ("/") so the files can be read correctly
	FileFolderPath string
}

type MongoStorageEntry struct {
	configuration *MongoStorageConfiguration
	database      *mgo.Database
	Id            bson.ObjectId `bson:"_id,omitempty"`
	Author        string `bson:"author,omitempty"`
	Filename      string `bson:"filename,omitempty"`
	ContentType   string `bson:"content_type,omitempty"`
	LastModified  time.Time `bson:"last_modified,omitempty"`
	UploadDate    time.Time `bson:"upload_date,omitempty"`
}

//<editor-fold desc="[MongoStorageEntry] Getter and Setter methods">
func (mongoStorageEntry *MongoStorageEntry) GetId() string {
	return mongoStorageEntry.Id.Hex()
}

func (mongoStorageEntry *MongoStorageEntry) GetAuthor() string {
	return mongoStorageEntry.Author
}

func (mongoStorageEntry *MongoStorageEntry) SetAuthor(author string) {
	mongoStorageEntry.Author = author
}

func (mongoStorageEntry *MongoStorageEntry) GetFilename() string {
	return mongoStorageEntry.Filename
}

func (mongoStorageEntry *MongoStorageEntry) SetFilename(filename string) {
	mongoStorageEntry.Filename = filename
}

func (mongoStorageEntry *MongoStorageEntry) GetContentType() string {
	return mongoStorageEntry.ContentType
}

func (mongoStorageEntry *MongoStorageEntry) SetContentType(contentType string) {
	mongoStorageEntry.ContentType = contentType
}

func (mongoStorageEntry *MongoStorageEntry) GetLastModifiedValue() time.Time {
	return mongoStorageEntry.LastModified
}

func (mongoStorageEntry *MongoStorageEntry) SetLastModifiedValue(lastModified time.Time) {
	mongoStorageEntry.LastModified = lastModified
}

func (mongoStorageEntry *MongoStorageEntry) GetUploadDate() time.Time {
	return mongoStorageEntry.UploadDate
}

func (mongoStorageEntry *MongoStorageEntry) SetUploadDate(uploadDate time.Time) {
	mongoStorageEntry.UploadDate = uploadDate
}

//</editor-fold>

//<editor-fold desc="[MongoStorageEntry] Storage methods">
func (mongoStorageEntry *MongoStorageEntry) Save() error {
	collection := mongoStorageEntry.database.C(mongoStorageEntry.configuration.UploadCollectionName)
	for {
		mongoStorageEntry.Id = bson.NewObjectId()
		if count, err := collection.FindId(mongoStorageEntry.Id).Limit(1).Count(); err != nil {
			return err
		} else if count >= 1 {
			continue
		} else {
			break
		}
	}
	err := collection.Insert(mongoStorageEntry)
	return err
}

func (mongoStorageEntry *MongoStorageEntry) Update() error {
	collection := mongoStorageEntry.database.C(mongoStorageEntry.configuration.UploadCollectionName)
	return collection.UpdateId(mongoStorageEntry.Id, mongoStorageEntry)
}

func (mongoStorageEntry *MongoStorageEntry) Delete() error {
	collection := mongoStorageEntry.database.C(mongoStorageEntry.configuration.UploadCollectionName)
	return collection.RemoveId(mongoStorageEntry.Id)
}

func (mongoStorageEntry *MongoStorageEntry) GetReadSeeker() (io.ReadSeeker, error) {
	path := mongoStorageEntry.configuration.FileFolderPath + mongoStorageEntry.GetId()
	return os.Open(path)
}

func (mongoStorageEntry *MongoStorageEntry) GetWriter() (io.WriteCloser, error) {
	path := mongoStorageEntry.configuration.FileFolderPath + mongoStorageEntry.GetId()
	return os.Create(path)
}

//</editor-fold>

type MongoStorage struct {
	DialInfo      *mgo.DialInfo
	Configuration *MongoStorageConfiguration
	session       *mgo.Session
	database      *mgo.Database
}

//<editor-fold desc="[MongoStorage] Storage methods">
func (mongoStorage *MongoStorage) Initialize() error {
	var err error
	if mongoStorage.session, err = mgo.DialWithInfo(mongoStorage.DialInfo); err != nil {
		return fmt.Errorf("An error occurred while connecting to MongoDB remote server: %v", err)
	} else {
		mongoStorage.database = mongoStorage.session.DB(mongoStorage.Configuration.DatabaseName)
		return nil
	}
}

func (mongoStorage *MongoStorage) Close() (bool, error) {
	mongoStorage.session.Close()
	return true, nil
}

func (mongoStorage *MongoStorage) NewStorageEntry() sharexhandler.Entry {
	uploadDate := time.Now()
	return &MongoStorageEntry{
		configuration: mongoStorage.Configuration,
		database:      mongoStorage.database,
		LastModified:  uploadDate,
		UploadDate:    uploadDate,
	}
}

func (mongoStorage *MongoStorage) LoadStorageEntry(id string) (bool, error, sharexhandler.Entry) {
	result := &MongoStorageEntry{}
	collection := mongoStorage.database.C(mongoStorage.Configuration.UploadCollectionName)
	if bson.IsObjectIdHex(id) {
		err := collection.FindId(bson.ObjectIdHex(id)).One(result)
		if err == mgo.ErrNotFound {
			return false, nil, result
		} else {
			result.configuration = mongoStorage.Configuration
			result.database = mongoStorage.database
			return err == nil, err, result
		}
	} else {
		return false, nil, result
	}
}

//</editor-fold>
