package notifications

type Message struct {
	Template  string
	Data      map[string]interface{}
	FromName  string
	FromEmail string
	FromPhone string
	ToName    string
	ToEmail   string
	ToPhone   string
	Subject   string
}
