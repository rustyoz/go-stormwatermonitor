package main

import (
	"encoding/gob"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"time"

	"github.com/GeertJohan/go.rice"
	"github.com/bmizerany/pat"
	geo "github.com/rustyoz/golang-geo"
)

var templates *template.Template

var t Tracker
var logging bool
var dir string
var radius int
var data string
var console bool
var maxproc int
var port int

var siteConfig SiteConfig

func init() {
	flag.StringVar(&dir, "dir", "", "data directory")
	flag.BoolVar(&logging, "log", false, "enable http request logging")
	flag.IntVar(&radius, "join", 0, "join radius in meters")
	flag.StringVar(&data, "data", "", "filename of preprocessed data file")
	flag.BoolVar(&console, "console", false, "enable console")
	flag.IntVar(&maxproc, "maxproc", runtime.NumCPU(), "set maximum processors")
	flag.StringVar(&siteConfig.BaseUrl, "baseurl", "", "Set base url of server")
	flag.IntVar(&port, "port", 9000, "port to bind too")
}

func main() {
	flag.Parse()
	fmt.Println("GOMAXPROCS", maxproc)
	runtime.GOMAXPROCS(maxproc)
	t.JoinRadius = radius
	fmt.Println("Reading Folder: ", dir)
	if logging {
		fmt.Println("HTTP request logging enabled")
	}
	if radius > 0 {
		fmt.Println("Joining points closer than: ", radius, " meters")
	}

	if filepath.Ext(data) == ".gob" {
		datafile, err := os.Open(data)
		if err != nil {
			fmt.Println(err)
			return
		}
		decoder := gob.NewDecoder(datafile)
		decoder.Decode(&t)
		datafile.Close()

	} else {
		err := t.OpenFolder(dir, radius)
		if err != nil {
			fmt.Println(err)
			return
		}
		datafile, err := os.Create("tracker.gob")
		if err != nil {
			fmt.Println(err)
			fmt.Println("Failed to create tracker.gob")
			return
		}

		encoder := gob.NewEncoder(datafile)
		err = encoder.Encode(t)
		if err != nil {
			fmt.Println(err)
			fmt.Println("Failed to encode data to tracker.gob")
		}
		datafile.Close()
	}

	//t.Open("small_subset_drains.shp")
	//t.Open("points.shp")
	//	t.Open(`council drain pipes.shp`)

	templates, _ = template.New("header").Parse(header)

	templates.New("body").Parse(body)

	mux := pat.New()

	mux.Get(siteConfig.BaseUrl+"/static/", http.StripPrefix(siteConfig.BaseUrl+"/static/", http.FileServer(rice.MustFindBox("static").HTTPBox())))
	mux.Get(siteConfig.BaseUrl+"/track", http.HandlerFunc(trackHandler))
	mux.Get(siteConfig.BaseUrl+"/", http.HandlerFunc(defaultHandler))

	http.Handle("/", mux)

	//fmt.Println(t.FindPathID(0))

	if console {
		fmt.Println("Console Enabled")
		go Console()
	}
	addr := ":" + strconv.Itoa(port)
	if logging {
		fmt.Println("Listening and Logging on " + addr)
		log.Fatal(http.ListenAndServe(addr, LogRequest(http.DefaultServeMux)))
	} else {
		fmt.Println("Listening on " + addr)
		http.ListenAndServe(addr, nil)
	}

}

func defaultHandler(w http.ResponseWriter, req *http.Request) {
	fmt.Println(req.URL)
	templates.ExecuteTemplate(w, "header", siteConfig)
	fmt.Fprintf(w, "%s", mapapi)

	templates.ExecuteTemplate(w, "body", siteConfig)
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

	var path []int
	var nearestid int
	var nearest *geo.Point
	nearpoints := FindAllStartPointsWithin(point, t.Points, float64(t.JoinRadius)/1000.0, &t)
	fmt.Println("Start Points within ", t.JoinRadius)
	fmt.Println(nearpoints)

	nearestid, nearest, distance = t.FindNearestStartPoint(point, distance)
	path = append(path, nearestid)
	path = append(path, t.FindPathID(nearestid)...)
	fmt.Println(nearestid)
	fmt.Println(nearest)
	path, _ = t.FindPathIDRecursive(path, 0, 0)

	//fmt.Println(path[len(path)-1])
	//fmt.Println(t.Points[path[len(path)-1]])
	/*if foundpath == true {
		fmt.Println(path)
	} else {
		fmt.Println("No path found")
	} */
	fmt.Println(path)

	points := t.IDsToPoints(path)
	geojson, err := t.PathToGeoJSON(points)
	fmt.Fprintf(w, "%s", geojson) // returns geojson to client

	handletime := time.Since(start)
	fmt.Println("Handled in: ", handletime)
}

func LogRequest(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Println(r.URL)
		handler.ServeHTTP(w, r)
	})
}

type SiteConfig struct {
	BaseUrl string
}
