// Public Domain (-) 2018-present, The Espian Source Authors.
// See the Espian Source UNLICENSE file for details.

package main

import (
	"net/http"

	"google.golang.org/appengine"
)

func handle(w http.ResponseWriter, r *http.Request) {
	if !appengine.IsDevAppServer() {
		if r.Host != "espians.com" || r.URL.Scheme != "https" {
			w.Header().Set("Location", "https://espians.com")
			w.WriteHeader(http.StatusMovedPermanently)
			return
		}
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
	}
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Write([]byte(`<!doctype html>
<meta charset=utf-8>
<title>Espians</title>
<meta name="viewport" content="width=device-width, initial-scale=1, viewport-fit=cover">
<script async src="https://www.googletagmanager.com/gtag/js?id=UA-90176-9"></script>
<script>
  dataLayer = [];
  function gtag(){dataLayer.push(arguments);}
  gtag('js', new Date());
  gtag('config', 'UA-90176-9');
</script>
<link rel="stylesheet" href="//fonts.googleapis.com/css?family=Montserrat%7CRaleway">
<style>
html {
	margin: 0;
	min-height: 100%;
	padding: 0;
}
body {
	background: #f50000;
	min-height: 100%;
	margin: 0;
	padding: 0;
}
.header {
	background: #f5f5f5;
	height: 60px;
	width: 100%;
}
.home {
	width: 100%;
}
.inner {
	width: 900px;
	margin: 0 auto;
}
.join {
	padding-top: 30px;
	text-align: center;
}
.join a {
	background: #2b2b2b;
	border: 1px solid #2b2b2b;
	border-radius: 5px;
	color: #fff;
	display: inline-block;
	font-family: Raleway,sans-serif;
	font-size: 30px;
	padding: 15px 18px;
	text-decoration: none;
}
.join a:hover, .join a:visited {
	text-decoration: none;
}
.join a:hover {
	border: 1px solid #fff;
}
.logo-image {
	float: left;
	margin: 11px 7px 0px 0px;
	text-decoration: none;
}
.logo-image img {
	height: 38px;
	width: 38px;
}
.logo-title {
	color: #2b2b2b;
	float: left;
	font-family: Montserrat,sans-serif;
	font-size: 17px;
	letter-spacing: 0.8px;
	margin-top: 20px;
	text-decoration: none;
}
.logo-title:hover, .logo-title:visited {
	text-decoration: none;
}
.tagline {
	color: #fff;
	font-family: Raleway,sans-serif;
	font-size: 30px;
	line-height: 42px;
	padding: 24px 0px;
}
.world-map {
	width: 100%;
}
@media screen and (max-width: 900px) {
	.inner {
		width: 100%;
	}
	.logo-image {
		margin-left: 14px;
	}
	.tagline {
		padding: 24px 20px;
	}
}
</style>
<div class="header"><div class="inner">
	<a href="/" class="logo-image"><img src="/.static/logo.png" alt="Espian Logo"></a>
	<a href="/" class="logo-title">ESPIANS</a>
</div></div>
<div class="home"><div class="inner">
	<div class="tagline">
		We are building the foundations for a decentralised society.
	</div>
	<img src="/.static/map.svg" class="world-map" alt="World Map">
	<div class="join">
		<a href="https://housing2.com">Join us @ Housing 2.0</a>
	</div>
</div></div>
`))
}

func main() {
	http.HandleFunc("/", handle)
	appengine.Main()
}
