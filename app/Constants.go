package app

//Language Enum to be used for languges
type Language int

//Topic Enum to be used for topics
type Topic int

//Status status of the game
type Status int

const (
	//LastGameIDKey used for the game
	LastGameIDKey = "last_game_id_key"
	//NumOfQuestionsInGame number of questions in a game
	NumOfQuestionsInGame = 10
)

const (
	//Active status of the game
	Active Status = iota + 1
	//Disconnected status of the game
	Disconnected
	//Finished status of the game
	Finished
)

const (
	//English default language
	English Language = iota + 1
	//Hindi language
	Hindi
	//Bengali LanguageD
	Bengali
	//Tamil Language
	Tamil
	//Odia Language
	Odia
)

const (
	//India default topic
	India Topic = iota + 1
	//Bollywood topic
	Bollywood
	//Science topic
	Science
	//Technology topic
	Technology
	//World topic
	World
)

func (l Language) String() string {
	return []string{"Default", "English", "Hindi", "Bengali", "Tamil", "Odia"}[l]
}

func (t Topic) String() string {
	return []string{"Default", "India", "Bollywood", "Science", "Technology", "World"}[t]
}

func (s Status) String() string {
	return []string{"Default", "Active", "Disconnected", "Finished"}[s]
}
