package server

import (
	"encoding/xml"
	"net/http"
	"os"
	"path"
	"strings"
)

// WebDAV XML types for Multi-Status responses.
type multistatus struct {
	XMLName   xml.Name   `xml:"DAV: multistatus"`
	Responses []response `xml:"response"`
}

type response struct {
	Href     string     `xml:"href"`
	Propstat []propstat `xml:"propstat"`
}

type propstat struct {
	Prop   prop   `xml:"prop"`
	Status string `xml:"status"`
}

type prop struct {
	DisplayName     string  `xml:"displayname,omitempty"`
	GetContentLength int64  `xml:"getcontentlength,omitempty"`
	GetLastModified string  `xml:"getlastmodified,omitempty"`
	ResourceType    resType `xml:"resourcetype"`
	GetContentType  string  `xml:"getcontenttype,omitempty"`
}

type resType struct {
	Collection *collection `xml:"collection,omitempty"`
}

type collection struct{}

// handleWebDAV routes WebDAV requests based on HTTP method.
func (s *Server) handleWebDAV(w http.ResponseWriter, r *http.Request) {
	// Map URL path /dav/Files/subdir to VFS path /Files/subdir.
	vfsPath := strings.TrimPrefix(r.URL.Path, "/dav")
	if vfsPath == "" {
		vfsPath = "/"
	}

	switch r.Method {
	case "OPTIONS":
		s.handleDavOptions(w, r)
	case "PROPFIND":
		s.handlePropfind(w, r, vfsPath)
	case "GET":
		s.handleDavGet(w, r, vfsPath)
	case "PUT":
		s.handleDavPut(w, r, vfsPath)
	default:
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
	}
}

// handleDavOptions responds to OPTIONS requests with supported DAV capabilities.
func (s *Server) handleDavOptions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("DAV", "1")
	w.Header().Set("Allow", "OPTIONS, PROPFIND, GET, PUT")
	w.WriteHeader(http.StatusOK)
}

// handlePropfind handles PROPFIND requests (Depth: 0 or 1).
func (s *Server) handlePropfind(w http.ResponseWriter, r *http.Request, vfsPath string) {
	depth := r.Header.Get("Depth")
	if depth == "" {
		depth = "infinity" // default, but we only support 0 and 1
	}

	ms := multistatus{}

	if depth == "0" {
		ms.Responses = append(ms.Responses, s.buildPropfindResponse(vfsPath))
	} else {
		// Depth 1: include directory contents.
		ms.Responses = append(ms.Responses, s.buildPropfindResponse(vfsPath))

		entry, err := s.vfs.ListDir(vfsPath)
		if err == nil {
			for _, f := range entry.Files {
				childPath := path.Join(vfsPath, f.Name)
				ms.Responses = append(ms.Responses, s.buildPropfindResponse(childPath))
			}
		}
		// If depth > 1 or infinity, we fall back to depth 1 behavior.
	}

	w.Header().Set("Content-Type", "application/xml; charset=utf-8")
	w.WriteHeader(http.StatusMultiStatus)
	xml.NewEncoder(w).Encode(ms)
}

// buildPropfindResponse builds a single PROPFIND response element for a VFS path.
func (s *Server) buildPropfindResponse(vfsPath string) response {
	localPath, _, err := s.vfs.GetFilePath(vfsPath)

	href := path.Join("/dav", vfsPath)

	ps := propstat{Status: "HTTP/1.1 200 OK"}

	if err != nil {
		ps.Status = "HTTP/1.1 404 Not Found"
		return response{Href: href, Propstat: []propstat{ps}}
	}

	info, err := os.Stat(localPath)
	if err != nil {
		ps.Status = "HTTP/1.1 404 Not Found"
		return response{Href: href, Propstat: []propstat{ps}}
	}

	ps.Prop.DisplayName = info.Name()
	ps.Prop.GetLastModified = info.ModTime().UTC().Format(http.TimeFormat)

	if info.IsDir() {
		// Directories must have a trailing slash for Windows Explorer compatibility.
		ps.Prop.ResourceType = resType{Collection: &collection{}}
		if !strings.HasSuffix(href, "/") {
			href += "/"
		}
	} else {
		ps.Prop.GetContentLength = info.Size()
		ps.Prop.GetContentType = detectMIME(info.Name())
	}

	return response{Href: href, Propstat: []propstat{ps}}
}

// handleDavGet handles GET requests for file downloads via WebDAV.
func (s *Server) handleDavGet(w http.ResponseWriter, r *http.Request, vfsPath string) {
	localPath, _, err := s.vfs.GetFilePath(vfsPath)
	if err != nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	info, err := os.Stat(localPath)
	if err != nil || info.IsDir() {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	mimeType := detectMIME(info.Name())
	w.Header().Set("Content-Type", mimeType)
	http.ServeFile(w, r, localPath)
}

// handleDavPut handles PUT requests for file uploads via WebDAV.
func (s *Server) handleDavPut(w http.ResponseWriter, r *http.Request, vfsPath string) {
	if vfsPath == "/" || vfsPath == "" {
		http.Error(w, "Cannot PUT to root", http.StatusBadRequest)
		return
	}

	if err := s.vfs.UploadFile(vfsPath, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
}
