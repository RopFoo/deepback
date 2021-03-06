package main

import (
	"context"
	"deepback/helper"
	"deepback/models"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var letters []models.Letter

var allQuestionsResponses []models.AllQuestionsResponse

var users []models.User

func getUser(userID string) models.User {
	var user models.User
	for _, u := range users {
		if user.ID.String() == userID {
			user = u
			break
		}
	}
	return user
}

func getLetters(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	//ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)

	collection := helper.ConnectToDB()

	cur, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		fmt.Println(err)
	}

	defer cur.Close(context.Background())

	users = nil

	for cur.Next(context.Background()) {

		// create a value into which the single document can be decoded
		var user models.User

		// & character returns the memory address of the following variable.
		err := cur.Decode(&user) // decode similar to deserialize process.
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(cur.Current)
		// add item our array
		users = append(users, user)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}

	allQuestionsResponses = nil
	//create array of all questions
	for _, user := range users {
		for _, question := range user.Questions {
			var allQuestionsResponse = models.AllQuestionsResponse{
				DisplayName: user.DisplayName,
				Title:       question.Title,
				Body:        question.Body,
				ID:          question.ID,
			}
			allQuestionsResponses = append(allQuestionsResponses, allQuestionsResponse)
		}
	}

	json.NewEncoder(w).Encode(allQuestionsResponses) // encode similar to serialize process.
}

func getQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var answerUser models.AnswerUser

	e := json.NewDecoder(r.Body).Decode(&answerUser)
	if e != nil {
		fmt.Println(e)
	}

	// init default user
	var user models.User

	// get id from url
	var params = mux.Vars(r)

	// save ID as 'MongoID'
	id, _ := primitive.ObjectIDFromHex(params["id"])

	// connect to MongoDB
	collection := helper.ConnectToDB()

	// create filtertype: find question by ID in db
	filter := bson.M{"questions": bson.M{"$elemMatch": bson.M{"_id": id}}}

	// save user with question in default user
	err := collection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		fmt.Println(err)
	}

	// init default question
	var question *models.Question

	// find question in user questions array
	for _, q := range user.Questions {
		if id == q.ID {
			question = q
		}
	}

	// check if its the users own question
	if user.Name == answerUser.UserID {
		fmt.Println("thats my own question")
		var stats models.Stats
		stats.Question = question
		stats.CalcMoods()
		json.NewEncoder(w).Encode(stats)
	} else {

		// check if user already send answer
		answered := false

		// if already answered - save answer for response
		var answerUserResponse models.AnswerUserResponse
		for _, a := range question.Answers {
			if answerUser.UserID == a.UserID {
				fmt.Println("User in Answer:", a.UserID)
				answerUserResponse.Answer = a
				answerUserResponse.Message = "already answered"
				answered = true
				break
			}
		}

		// return question
		if answered == false {
			var questionResponse = models.QuestionResponse{
				ID:    question.ID,
				Title: question.Title,
				Body:  question.Body,
			}
			json.NewEncoder(w).Encode(questionResponse)
		} else {
			json.NewEncoder(w).Encode(answerUserResponse)
		}
	}

}

func sendAnswer(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var letter models.Letter

	err := json.NewDecoder(r.Body).Decode(&letter)
	if err != nil {
		fmt.Println(err)
	}

	collection := helper.ConnectToDB()

	questionID, _ := primitive.ObjectIDFromHex(letter.QuestionID)

	array := bson.M{"questions": bson.M{"$elemMatch": bson.M{"_id": questionID}}}
	pushToArray := bson.M{
		"$push": bson.M{
			"questions.$.answers": bson.M{
				"mood":   letter.Mood,
				"title":  letter.Title,
				"body":   letter.Body,
				"userID": letter.UserID,
				"_id":    primitive.NewObjectID()}}}

	collection.UpdateOne(context.TODO(), array, pushToArray)

}

func postQuestion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var question models.Question

	err := json.NewDecoder(r.Body).Decode(&question)
	if err != nil {
		fmt.Println(err)
	}

	collection := helper.ConnectToDB()

	//userID, _ := primitive.ObjectIDFromHex(question.UserID)

	filter := bson.M{"name": question.UserID}

	var user models.User

	e := collection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		fmt.Println(e)
	}

	fmt.Printf("Found a single document: %+v\n", user.ID)

	// if user exists -> post question to questions
	// else -> create User and post question to questions

	defaultID, _ := primitive.ObjectIDFromHex("000000000000000000000000")

	if question.UserName == "" {
		fmt.Println("emtpy user")
	} else if user.ID == defaultID {
		fmt.Println("User doesn't exist!")
		user.Name = question.UserID
		user.DisplayName = question.UserName
		insertResult, err := collection.InsertOne(context.TODO(), user)
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("created User: ", insertResult)

		pushToArray := bson.M{"$push": bson.M{"questions": bson.M{"title": question.Title, "body": question.Body, "_id": primitive.NewObjectID()}}}
		collection.UpdateOne(context.TODO(), filter, pushToArray)

	} else {
		pushToArray := bson.M{"$push": bson.M{"questions": bson.M{"title": question.Title, "body": question.Body, "_id": primitive.NewObjectID()}}}
		collection.UpdateOne(context.TODO(), filter, pushToArray)
	}

}

func getUserQuestions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var user models.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		fmt.Println(err)
	}

	collection := helper.ConnectToDB()

	filter := bson.M{"name": user.Name}

	e := collection.FindOne(context.TODO(), filter).Decode(&user)
	if err != nil {
		fmt.Println(e)
	}

	json.NewEncoder(w).Encode(user.Questions)

}

func getUserAnswers(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	var user models.User

	err := json.NewDecoder(r.Body).Decode(&user)
	if err != nil {
		fmt.Println(err)
	}

	collection := helper.ConnectToDB()

	//filter := bson.M{"name": user.Name}

	fmt.Println(user.Name)
	filter := bson.M{"questions": bson.M{"$elemMatch": bson.M{"answers": bson.M{"$elemMatch": bson.M{"userID": user.Name}}}}}

	cur, err := collection.Find(context.Background(), filter)
	if err != nil {
		fmt.Println(err)
	}

	defer cur.Close(context.Background())

	users = nil

	for cur.Next(context.Background()) {

		// create a value into which the single document can be decoded
		var user models.User

		// & character returns the memory address of the following variable.
		err := cur.Decode(&user) // decode similar to deserialize process.
		if err != nil {
			log.Fatal(err)
		}
		//fmt.Println(cur.Current)
		// add item our array
		users = append(users, user)
	}

	if err := cur.Err(); err != nil {
		log.Fatal(err)
	}
	var answers []models.AnswerFromUser
	var answ models.AnswerFromUser

	for _, u := range users {
		for _, question := range u.Questions {
			for _, answer := range question.Answers {
				if answer.UserID == user.Name {
					answ.Author = u.Name
					answ.Answer = answer
					answ.QuestionID = question.ID
					answ.Title = question.Title
					answ.Body = question.Body
					answers = append(answers, answ)
				}
			}
		}
	}

	fmt.Println(answers)

	json.NewEncoder(w).Encode(answers)

}

func main() {

	router := mux.NewRouter()

	router.HandleFunc("/api/letters", getLetters).Methods("GET")
	router.HandleFunc("/api/question/{id}", getQuestion).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/letters", sendAnswer).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/ask", postQuestion).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/user-questions", getUserQuestions).Methods("POST", "OPTIONS")
	router.HandleFunc("/api/user-answers", getUserAnswers).Methods("POST", "OPTIONS")

	log.Fatal(http.ListenAndServe(":8000", router))

}
