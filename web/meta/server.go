// Public Domain (-) 2018-present, The Espian Source Authors.
// See the Espian Source UNLICENSE file for details.

package main

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"strings"
	"time"

	"google.golang.org/appengine"
	"google.golang.org/appengine/log"
	"google.golang.org/appengine/user"
)

// AuthToken is used by CLI applications.
type AuthToken struct {
	Created time.Time
	Label   string
	Revoked bool
	User    string
}

// Cluster represents a set of deployment nodes.
type Cluster struct {
	BootToken []byte
	ProjectID string
	WebHost   string
}

// Config for the meta server.
type Config struct {
	Admins   map[string]bool
	Clusters map[string]*Cluster
	Server   string
	Users    map[string]bool
}

func handle(w http.ResponseWriter, r *http.Request) {

	if !appengine.IsDevAppServer() {
		if r.Host != config.Server || r.URL.Scheme != "https" {
			w.Header().Set("Location", "https://"+config.Server)
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}

	ctx := appengine.NewContext(r)
	path := r.URL.Path

	if path == "/logout" {
		url, err := user.LogoutURL(ctx, "/")
		if err != nil {
			log.Errorf(ctx, "could not get logout url: %v", err)
			serverError(w)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}

	if r.Method == "POST" {
		if path == "/node.register" {
			return
		}

		if strings.HasPrefix(path, "/node/") {
			switch path {
			case "ping":
			case "list":
			default:
				http.NotFound(w, r)
			}
			return
		}
	}

	if strings.HasPrefix(path, "/cli/") {
		query := r.URL.Query()
		app := query.Get("app")
		token := query.Get("token")
		_ = app
		_ = token
		switch path[5:] {
		case "deploy":
			return
		case "upload":
			return
		case "promote":
			return
		default:
			http.NotFound(w, r)
		}
		return
	}

	u := user.Current(ctx)
	if u == nil {
		url, err := user.LoginURL(ctx, r.URL.String())
		if err != nil {
			log.Errorf(ctx, "could not get login url: %v", err)
			serverError(w)
			return
		}
		w.Header().Set("Location", url)
		w.WriteHeader(http.StatusFound)
		return
	}

	if !config.Users[u.Email] {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(
			"<h1>Not Authorised</h1><a href='/logout'>Log out " +
				html.EscapeString(u.Email) +
				"</a>",
		))
		return
	}

	switch path {
	case "/":
		// List deployments
		w.Write([]byte("<h1>Meta Server</h1>"))
	case "/token.create":
		// Create auth token
		q := r.URL.Query()
		port, err := strconv.ParseInt(q.Get("port"), 10, 64)
		if err != nil {
			log.Errorf(ctx, "got invalid port value: %s", q.Get("port"))
			serverError(w)
			return
		}
		// label := q.Get("label")
		w.Header().Set("Location", fmt.Sprintf("http://127.0.0.1:%d/?token=", port))
		w.WriteHeader(http.StatusFound)
	case "/token.revoke":
		// Mark token as revoked
		// If Admin, enable for all tokens
		// CSRF
	case "/tokens":
		// List tokens
		// If Admin, show all tokens
	default:
		http.NotFound(w, r)
	}

}

func serverError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("<h1>Internal Server Error</h1>"))
}

func verifyAuthToken() {}

func main() {
	http.HandleFunc("/", handle)
	appengine.Main()
}
