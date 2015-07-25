package main

const body = `
<body>
<nav class="navbar navbar-inverse navbar-fixed-top">
<div class="container">
<div class="navbar-header">
<button type="button" class="navbar-toggle collapsed" data-toggle="collapse" data-target="#navbar" aria-expanded="false" aria-controls="navbar">
<span class="sr-only">Toggle navigation</span>
<span class="icon-bar"></span>
<span class="icon-bar"></span>
<span class="icon-bar"></span>
</button>
<a class="navbar-brand" href="/stormwatermonitor/">Mighty Rubber Duck - the pollution tracker system</a>
</div>
<div id="navbar" class="collapse navbar-collapse">
<ul class="nav navbar-nav">
<li class="active"><a href="#">Home</a></li>
<li class="active"><a href="https://hackerspace.govhack.org/content/mighty-ducks-stormwater-tracker">About</a></li>
</ul>
</div><!--/.nav-collapse -->
</div>
</nav>

<div class="container">
<div class="panel">
<div class="subp_left">
</div>
<div class="subp_right">
<div id="googleMap" style="width:60vw;height:60vh;"></div>
</div>
<div class="infoDisplay">
<div class="subp_bottom">
<p id='description'>After selecting your location, click to button below</p>
<form id="point">
Latitude: <input type="text" id= "lat" class="coordiantes" value="Drag the pin" name="lat"><br>
Longitude: <input type="text" id="lng" class="coordinates" value="Drag the pin" name="lng"><br>
<p>What do you put down the storm water drain?</p>
<select name="spill" class="coordinates">
<option value="detergent">Detergent</option>
<option value="oil">Oil</option>
<option value="duck">Rubber Ducky</option>
</select>
<input type="submit" value="Submit">
</form>
</div>
`

const header = `
<head>
<meta name="viewport" content="width=device-width, initial-scale=1">

<meta></meta>
<script src="https://ajax.googleapis.com/ajax/libs/jquery/2.1.3/jquery.min.js"></script>
<script src="http://maps.googleapis.com/maps/api/js"></script>
<link href="{{.BaseUrl}}/static/css/bootstrap.min.css" rel="stylesheet">
<link href="{{.BaseUrl}}/static/css/style.css" rel="stylesheet">

<!--Deakin Lng -38.144061, Lat 144.360345-->
`

const mapapi = `
<script>
	var map;
	//var maker = new google.maps.Marker({map:map, });
	//var infowindow = new google.maps.InfoWindow();
	var marker;
	var infowindow;
	var fLng, fLat;
	var myCenter = new google.maps.LatLng(-38.144061, 144.360345); //lag and lng
	//var myCenter0;
	var failed = false;

	function locate(){
		navigator.geolocation.getCurrentPosition(initialize, fail);
	}

	function fail(){
		failed = true;
		initialize()
	}
	function initialize(position) {
	//function initialize(){

		if (failed == false) {
			//locating works
			myCenter = new google.maps.LatLng(position.coords.latitude, position.coords.longitude);
		}

		var mapProp = {
			center: myCenter,
			zoom: 17,
			mapTypeId: google.maps.MapTypeId.HYBRID
		};
		map=new google.maps.Map(document.getElementById("googleMap"), mapProp);
		//event click to drop a pin
		marker = new google.maps.Marker({
			map:map,
			position: myCenter,
			draggable: true,
			title: "Drag me!"
		})
		infowindow = new google.maps.InfoWindow({
				content: 'Latitude: ' + myCenter.lat() + '<br>Longtitude: ' + myCenter.lng()
			});

		google.maps.event.addListener(marker, 'drag', function(event){
			updateForm(event.latLng)
		});

	}

	function updateForm(location){
		fLng = document.getElementById("lng");
		fLat = document.getElementById("lat");
		fLng.value = location.lng();
		fLat.value = location.lat();
	}
	google.maps.event.addDomListener(window, 'load', locate);
	//$(#googleMap).load("static/loading2.gif");
	/*
	version 0.4 changelog:
	have finished the function that allows drag and drop marker and update the coordinates in the input field. thus these coordinates can be transferred to another page via GET method
	*/
</script>
`

const stylescript = `
<script>
var featureStyle = {
	strokeColor:"#0063BD",
	 strokeOpacity:0.8,
	 strokeWeight:5
}

</script>
`

const submitscript = `
<script>
// Attach a submit handler to the form
$( '#point' ).submit(function( event ) {

 var $form = $(this);
  // Stop form from submitting normally
  event.preventDefault();

  var postdata = $form.serialize();
  var posturl = 'track?' // $form.attr( "action" );

	map.data.forEach(function(feature) {
        //If you want, check here for some constraints.
        map.data.remove(feature);
    });
  map.data.loadGeoJson(posturl + postdata)
	map.data.setStyle(featureStyle);

});
</script>
`
