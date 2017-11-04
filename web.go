package coquelicot

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path"
	"time"
	"strings"
	"net/url"
	"path/filepath"

	//	"github.com/gin-gonic/gin"
	"github.com/pborman/uuid"
)

type H map[string]interface{}

func toJSON(w http.ResponseWriter, code int, obj interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	b, err := json.Marshal(obj)
	if err != nil {
		log.Println("error:", err.Error())
		return
	}
	if _, err := w.Write(b); err != nil {
		log.Println("error:", err.Error())
		return
	}
}

// ResumeHandler allows resuming a file upload.
//func (s *Storage) ResumeHandler(c *gin.Context) {
func (s *Storage) ResumeHandler(w http.ResponseWriter, r *http.Request) {
	status := http.StatusOK
	filename := r.URL.Query().Get("file")

	cookie, _ := r.Cookie("coquelicot")
	offset := int64(0)

	if cookie != nil {
		hasher := md5.New()
		hasher.Write([]byte(cookie.Value + filename))
		chunkname := hex.EncodeToString(hasher.Sum(nil))
		fi, err := os.Stat(path.Join(s.output, "chunks", chunkname))
		if err != nil {
			if !os.IsNotExist(err) {
				toJSON(w, http.StatusInternalServerError, H{
					"status": http.StatusText(http.StatusInternalServerError),
					"error":  fmt.Sprintf("Resume error: %q", err.Error()),
				})
				return
			}
		} else {
			offset = fi.Size()
		}
	}

	toJSON(w, status, H{"status": http.StatusText(status), "file": H{"size": offset}})
}

// UploadHandler is the endpoint for uploading and storing files.
//func (s *Storage) UploadHandler(c *gin.Context) {


func (s *Storage) UploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "GET" {
		status := http.StatusOK
		// FIXME: nil content
		toJSON(w, status, H{"status": http.StatusText(status), "files": nil})
		return
	}
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	converts, err := getConvertParams(r)
	if err != nil {
		toJSON(w, http.StatusBadRequest, H{
			"status": "error",
			"error":  fmt.Sprintf("Query params: %s", err),
		})
		return
	}
	converts["original"] = ""

	// File upload cookie so we can keep track of chunks.
	cookie, _ := r.Cookie("coquelicot")
	if cookie == nil {
		cookie = &http.Cookie{
			Name:    "coquelicot",
			Value:   uuid.New(),
			Expires: time.Now().Add(2 * 24 * time.Hour),
			Path:    "/",
		}
		r.AddCookie(cookie)
		http.SetCookie(w, cookie)
	}

	// Performs the processing of writing data into chunk files.
	files, err := process(r, s.StorageDir())

	if err == incomplete {
		toJSON(w, http.StatusOK, H{
			"status": http.StatusText(http.StatusOK),
			"file":   H{"size": files[0].Size},
		})
		return
	}
	if err != nil {
		toJSON(w, http.StatusBadRequest, H{
			"status": http.StatusText(http.StatusBadRequest),
			"error":  fmt.Sprintf("Upload error: %q", err.Error()),
		})
		return
	}

	data := make([]map[string]interface{}, 0)
	// Expected status if no error
	status := http.StatusCreated

	for _, ofile := range files {
		// true to delete final chunk
		//attachment, err := create(s.StorageDir(), ofile, converts, true)
		attachment, err := s.CreateAttach(ofile,converts,true)
		if err != nil {
			data = append(data, map[string]interface{}{
				"name":  ofile.Filename,
				"size":  ofile.Size,
				"error": err.Error(),
			})
			status = http.StatusInternalServerError
			continue
		}
		data = append(data, attachment.ToJson())
	}

	toJSON(w, status, H{"status": http.StatusText(status), "files": data})
}

// Get parameters for convert from Request query string
func getConvertParams(req *http.Request) (map[string]string, error) {
	raw_converts := req.URL.Query().Get("converts")

	if raw_converts == "" {
		raw_converts = "{}"
	}

	convert := make(map[string]string)

	err := json.Unmarshal([]byte(raw_converts), &convert)
	if err != nil {
		return nil, err
	}

	return convert, nil
}



func escape(s string) string {
  return strings.Replace(url.QueryEscape(s), "+", "%20", -1) 
}

func extractKey(r *http.Request) string {
  // Use RequestURI instead of r.URL.Path, as we need the encoded form:
  path := strings.Split(r.RequestURI, "?")[0]   // r.RequestURI => /delete?file=fuckyou...
  // Also adjust double encoded slashes:
  return strings.Replace(path[1:], "%252F", "%2F", -1) 
}

func check(err error) {
  if err != nil {
    panic(err)
  }
}

func (s *Storage) DeleteHandler(w http.ResponseWriter, r *http.Request) {
	/*
	if r.Method == "GET" {
		status := http.StatusOK
		// FIXME: nil content
		toJSON(w, status, H{"status": http.StatusText(status), "files": nil})
		return
	}

	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	
	if r.Method == "DELETE" {
		log.Printf("Start Delete")
	}
	*/
	key := extractKey(r)
  parts := strings.Split(key, "/")

	if len(parts) == 3 {
		result := make(map[string]bool, 1)
		key := parts[1]+"/"+ strings.Split(parts[2],".")[0]
		orig_file := s.StorageDir()+"/"+ key + filepath.Ext(parts[2])	
		thumbnail_file := s.StorageDir()+"/"+key+"-thumbnail"+filepath.Ext(parts[2])
		
		//delete original file
		if err := os.Remove(orig_file);err == nil {
		  result[key] = true
		}
		// delete thumbnail 
		if err := os.Remove(thumbnail_file);err == nil {
			result[thumbnail_file] = true
		}
		// delete the hash dir
		if err := os.Remove( s.StorageDir()+"/"+parts[1] ); err == nil {
			result[ parts[1] ]  = true 
		}

	  w.Header().Set("Content-Type", "application/json")
    b, err := json.Marshal(result)
    check(err)
    fmt.Fprintln(w, string(b))

	}else {
    http.Error(w, "405 Method not allowed", http.StatusMethodNotAllowed)
  }

}




