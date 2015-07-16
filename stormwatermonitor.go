package main

import (
	"fmt"
	"github.com/bmizerany/pat"
	"github.com/kellydunn/golang-geo"
	"html/template"
	"net/http"
	"strconv"
	//"strings"
)

var templates *template.Template

var t Tracker

func main() {

	//t.Open("small_subset_drains.shp")
	t.Open("points.shp")
	templates, _ = template.New("header").Parse(header)

	templates.New("body").Parse(body)

	mux := pat.New()

	mux.Get("/", http.HandlerFunc(defaultHandler))
	mux.Get("/track/", http.HandlerFunc(trackHandler))
	mux.Get("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	http.Handle("/", mux)

	fmt.Println(`http.ListenAndServe(":8000", nil)`)
	http.ListenAndServe(":8000", nil)

}

func defaultHandler(w http.ResponseWriter, req *http.Request) {

	templates.ExecuteTemplate(w, "header", nil)
	fmt.Fprintf(w, "%s", mapapi)
	fmt.Fprintf(w, "%s", submitscript)
	templates.ExecuteTemplate(w, "body", nil)

}

func trackHandler(w http.ResponseWriter, req *http.Request) {
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
	var path []geo.Point
	var nearestid int
	var nearest *geo.Point
	for foundpath == false {
		nearestid, nearest, distance = t.FindNearestPoint(point, distance)
		fmt.Println(nearestid)
		fmt.Println(nearest)
		path, foundpath = t.FindPath(nearestid)
	}
	if foundpath == true {
		fmt.Println(path)
	} else {
		fmt.Println("No path found")
	}

	geojson, err := t.PathToGeoJSON(path)
	fmt.Fprintf(w, "%s", geojson)

}
