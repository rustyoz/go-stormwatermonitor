package main

import (
	"fmt"
	"github.com/bmizerany/pat"
	//"github.com/davecheney/profile"
	"flag"
	"github.com/kellydunn/golang-geo"
	"html/template"
	"net/http"

	"github.com/GeertJohan/go.rice"
	"runtime"
	"strconv"
	"time"
	//"strings"
)

var templates *template.Template

var t Tracker
var log bool
var dir string

func init() {
	flag.StringVar(&dir, "dir", "", "data directory")
	flag.BoolVar(&log, "log", false, "enable http request logging")

}

func main() {
	runtime.GOMAXPROCS(runtime.NumCPU())
	fmt.Println("GOMAXPROCS", runtime.GOMAXPROCS(0))

	flag.Parse()
	fmt.Println(dir)
	fmt.Println("logging: ", log)
	err := t.OpenFolder(dir)
	if err != nil {
		fmt.Println(err)
		return
	}

	//defer profile.Start(profile.CPUProfile).Stop()
	//t.Open("small_subset_drains.shp")
	//t.Open("points.shp")
	//	t.Open(`council drain pipes.shp`)

	templates, _ = template.New("header").Parse(header)

	templates.New("body").Parse(body)

	mux := pat.New()

	mux.Get("/", http.HandlerFunc(defaultHandler))
	mux.Get("/track", http.HandlerFunc(trackHandler))
	mux.Get("/static/", http.StripPrefix("/static/", http.FileServer(rice.MustFindBox("static").HTTPBox())))
	http.Handle("/", mux)

	//fmt.Println(t.FindPathID(0))
	fmt.Println(`http.ListenAndServe(":8000", nil)`)

	if log {
		http.ListenAndServe(":8000", Log(http.DefaultServeMux))
	} else {
		http.ListenAndServe(":8000", nil)
	}

}

func defaultHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.URL)
	templates.ExecuteTemplate(w, "header", nil)
	fmt.Fprintf(w, "%s", mapapi)

	templates.ExecuteTemplate(w, "body", nil)
	fmt.Fprintf(w, "%s", stylescript)
	fmt.Fprintf(w, "%s", submitscript)
}

func trackHandler(w http.ResponseWriter, req *http.Request) {
	start := time.Now()
	queury := req.URL.Query()
	fmt.Println(req.URL.String())
	//fmt.Fprintf(w, "%s \n", queury.Get("lat"))
	//fmt.Fprintf(w, "%s \n", queury.Get("lng"))
	//	fmt.Fprintf(w, "%s \n", queury.Get("spill"))

	lat, err := strconv.ParseFloat(queury.Get("lat"), 64)
	lng, err := strconv.ParseFloat(queury.Get("lng"), 64)
	if err != nil {
		defaultHandler(w, req)
		return
	}
	point := geo.NewPoint(lat, lng)
	var distance float64
	distance = 0
	var foundpath bool
	var path []int
	var nearestid int
	var nearest *geo.Point
	for foundpath == false {
		nearestid, nearest, distance = t.FindNearestPoint(point, distance)
		fmt.Println(nearestid)
		fmt.Println(nearest)
		path, foundpath = t.FindPathID(nearestid)
	}
	/*if foundpath == true {
		fmt.Println(path)
	} else {
		fmt.Println("No path found")
	} */
	points := t.PathToPoints(path)
	geojson, err := t.PathToGeoJSON(points)
	fmt.Fprintf(w, "%s", geojson) // returns geojson to client

	handletime := time.Since(start)
	fmt.Println("Handled in: ", handletime)
}

func Log(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println(r.URL)
		handler.ServeHTTP(w, r)
	})
}
