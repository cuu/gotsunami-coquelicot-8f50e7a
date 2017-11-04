package coquelicot

import (
	"fmt"
	"os"
)

// attachment contain info about directory, base mime type and all files saved.
type attachment struct {
	Storage     string
	Host        string
	originalFile *originalFile
	Dir          *dirManager
	Versions     map[string]fileManager
}

// Function receive root directory, original file, convertion parameters.
// Return attachment saved. The final chunk is deleted if delChunk is true.

func (s *Storage) CreateAttach( ofile *originalFile, converts map[string]string, delChunk bool) (*attachment, error) {
	dm, err := createDir(s.StorageDir(), ofile.BaseMime)
	if err != nil {
		return nil, err
	}

	at := &attachment{
		Storage: s.StorageDir(),
		Host:    s.Host(),
		originalFile: ofile,
		Dir:          dm,
		Versions:     make(map[string]fileManager),
	}

	if ofile.BaseMime == "image" {
		converts["thumbnail"] = "120x90"
	}

	makeVersion := func(a *attachment, version, convert string) error {
		fm, err := at.createVersion(version, convert)
		if err != nil {
			return err
		}
		at.Versions[version] = fm
		return nil
	}

	if err := makeVersion(at, "original", ""); err != nil {
		return nil, err
	}

	if makeThumbnail {
		if err := makeVersion(at, "thumbnail", converts["thumbnail"]); err != nil {
			return nil, err
		}
	}

	if delChunk {
		return at, os.Remove(at.originalFile.Filepath)
	}
	return at, nil
}


func create(storage string, ofile *originalFile, converts map[string]string, delChunk bool) (*attachment, error) {
	dm, err := createDir(storage, ofile.BaseMime)
	if err != nil {
		return nil, err
	}

	at := &attachment{
		originalFile: ofile,
		Dir:          dm,
		Versions:     make(map[string]fileManager),
	}

	if ofile.BaseMime == "image" {
		converts["thumbnail"] = "120x90"
	}

	makeVersion := func(a *attachment, version, convert string) error {
		fm, err := at.createVersion(version, convert)
		if err != nil {
			return err
		}
		at.Versions[version] = fm
		return nil
	}

	if err := makeVersion(at, "original", ""); err != nil {
		return nil, err
	}

	if makeThumbnail {
		if err := makeVersion(at, "thumbnail", converts["thumbnail"]); err != nil {
			return nil, err
		}
	}

	if delChunk {
		return at, os.Remove(at.originalFile.Filepath)
	}
	return at, nil
}

// Directly save single version and return fileManager.
func (attachment *attachment) createVersion(version string, convert string) (fileManager, error) {
	fm := newFileManager(attachment.Dir, attachment.originalFile.BaseMime, version)
	fm.SetFilename(attachment.originalFile)

	if err := fm.convert(attachment.originalFile.Filepath, convert); err != nil {
		return nil, err
	}

	return fm, nil
}


func (attachment *attachment)  ImageUrl() string {
	return attachment.Host+"/image"
}


func (attachment *attachment) ToJson() map[string]interface{} {
	data := make(map[string]interface{})
	data["type"] = attachment.originalFile.BaseMime
	data["dir"] = attachment.Dir.Path
	data["name"] = attachment.originalFile.Filename
		
	versions := make(map[string]interface{})
	for version, fm := range attachment.Versions {
		versions[version] = fm.ToJson()
	}

	data["versions"] = versions
	
	// Blueimp jquery file upload patch
	v1,ok := versions["thumbnail"].(map[string]interface{})
	if ok {
		data["thumbnailUrl"] = attachment.ImageUrl()+ string(v1["url"].(string))
	}
	v1,ok = versions["original"].(map[string]interface{})
	if ok {
		data["url"]          =  attachment.ImageUrl()+v1["url"].(string)
		data["deleteUrl"]    =  attachment.Host+"/delete"+v1["url"].(string)
		data["size"]         = v1["size"].(int64)
		data["deleteType"]   = "DELETE"
	}

	fmt.Println(versions)
	return data
}
