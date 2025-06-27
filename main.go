package main
import("fmt"
 "encoding/json"
"net/http")

type dbase struct {
    Name     string  `json:"name"`
}
func submitHandler(w http.ResponseWriter, r *http.Request){
    w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	response := dbase{Name: "Hello from Go backend!"}
	json.NewEncoder(w).Encode(response)
}
func main(){
	fmt.Println("HELLO WROLS")
    http.HandleFunc("/api/submit", submitHandler)

	fmt.Println("Server running at http://localhost:8080")
	http.ListenAndServe(":8080", nil)

}