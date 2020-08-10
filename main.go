package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	reportParse "battlereportparsing"

	"github.com/gorilla/mux" // http router used

	// for .env variables compatability

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// SubmitRequest - The information recieved when revieving a post submit request.
type SubmitRequest struct {
	BattlePaste string `json:"battlePaste"`
}

// SubmitResponse - The information responded with when revieving a post submit request.
type SubmitResponse struct {
	MongoID string `json:"mongoId"`
	Server  string `json:"server"`
}

// SearchRequest - The information sent to the server when a user makes a search Battle Report request.
type SearchRequest struct {
	Server       string `json:"server"`
	Coordinate   string `json:"coordinate"`
	PlayerName   string `json:"playerName"`
	MinTotalLoss uint   `json:"minTotalLoss"`
}

//const urlReact = "https://seanb.herokuapp.com/battlepaste"

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(os.Getenv("ATLAS_URI")))

	if err != nil {
		log.Fatal(err)
	}
	db = client.Database("battle_reports")

	handler()
}

var db *mongo.Database

func handler() {
	port := os.Getenv("PORT")

	r := mux.NewRouter()
	r.HandleFunc("/battlereport/{server}/{reportID}", viewBattleReport).Methods(http.MethodGet, http.MethodOptions)
	r.HandleFunc("/battlereport/submit", submitBattleReport).Methods(http.MethodPost, http.MethodOptions)
	r.HandleFunc("/battlereport/search", searchBattleReport).Methods(http.Methodpost, http.MethodOptions)
	log.Fatal(http.ListenAndServe(":"+port, r)) // If error then log to console
}

//https://www.mongodb.com/blog/post/quick-start-golang--mongodb--modeling-documents-with-go-data-structures

func home(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "home")
}

func searchBattleReport(w http.ResponseWriter, r *http.Request) {
	if (*req).Method == "OPTIONS" {
		allowOpts(&w)
		return
	}

	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	var searchReq SearchRequest
	err = json.Unmarshal(b, &searchReq)
	if err != nil {
		fmt.Println(err)
		http.Error(w, err.Error(), 501)
		return
	}

	collection := db.Collection(strings.ToLower(searchReq.Server))

	// Check search criteria from user request and add it to search parameters is defined.
	var rules bson.D

	// Rule for if the attack or the defender contains the desired player name

	if len(searchReq.PlayerName) > 0 {
		rules = bson.D{
			{"$or", bson.A{
				bson.M{"Attacker.PlayerName": searchReq.PlayerName},
				bson.M{"Defender.PlayerName": searchReq.PlayerName},
			}},
		}
	}

	if len(searchReq.Coordinate) > 0 {
		rules = append(rules, bson.E{"Coordinate", searchReq.Coordinate})
	}

	rules = append(rules, bson.E{"TotalLoss", bson.D{{"$gt", searchReq.MinTotalLoss}}})

	opts := options.Find()
	opts.SetSort(bson.D{{"TotalLoss", -1}})

	cursor, err := collection.Find(context.TODO(), rules, opts)

	// Decending order based on total fleet loss
	var reportsSorted []bson.M
	if err = cursor.All(context.TODO(), &reportsSorted); err != nil {
		log.Fatal(err)
	}

	resObj, _ := json.Marshal(reportsSorted)
	allowOpts(&w)
	w.Write(resObj)
}

func viewBattleReport(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)

	server := vars["server"]
	id, _ := primitive.ObjectIDFromHex(vars["reportID"]) //strconv.Atoi(vars["reportID"])

	collection := db.Collection(strings.ToLower(server))

	var report reportParse.BattleReport
	err := collection.FindOne(context.TODO(), bson.M{"_id": id}).Decode(&report) //, options.FindOne().SetProjection(bson.M{"_id": 0}))

	//fmt.Printf("\n%+v\n", data)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	resObj, _ := json.Marshal(report)
	allowOpts(&w)
	w.Write(resObj)
}

func submitBattleReport(w http.ResponseWriter, r *http.Request) {
	b, err := ioutil.ReadAll(r.Body)
	defer r.Body.Close()
	if err != nil {
		log.Fatal(err)
	}

	var report SubmitRequest
	err = json.Unmarshal(b, &report)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	//report, _ := ioutil.ReadFile("report2.txt")
	battleReport, parseErr := reportParse.Parse(report.BattlePaste) //reportParse.Parse(r.Header.data)
	if parseErr {
		http.Error(w, "Error with parse", 505)
		return
	}

	collection := db.Collection(strings.ToLower(battleReport.Server))

	//ret, _ := json.Marshal(battleReport)
	id, _ := collection.InsertOne(context.TODO(), battleReport)
	hexID := (id.InsertedID).(primitive.ObjectID).Hex()

	var res SubmitResponse
	res.MongoID = hexID
	res.Server = battleReport.Server

	resObj, _ := json.Marshal(res)

	w.Header().Set("Content-Type", "application/json")
	allowOpts(&w)
	w.Write(resObj)
}

func allowOpts(w *http.ResponseWriter) {
	(*w).Header().Set("Access-Control-Allow-Origin", "*")
    (*w).Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
    (*w).Header().Set("Access-Control-Allow-Headers", "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization")
}
