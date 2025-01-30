# GoRigumi 🛠️  

![Go](https://img.shields.io/badge/Go-1.18%2B-blue)
A simple and powerful Go toolkit for handling file uploads, JSON processing, and other utility functions.

📌 **Repository:** [github.com/drunkleen/gorigumi](https://github.com/drunkleen/gorigumi)

---

## Features 🚀  

✅ File Upload (single & multiple)  
✅ Secure Random String Generation  
✅ File Download Handling  
✅ JSON Processing (Read, Write, Error Handling)  
✅ Slug Conversion for URLs  
✅ Directory Creation Utility  

---

## Installation 💻  

```sh
go get github.com/drunkleen/gorigumi
```

---

## Usage 📖  

### 1️⃣ Initialize the GoRigumi  

```go
package main

import (
	"fmt"
	"github.com/drunkleen/gorigumi"
)

func main() {
	gorigumi := gorigumi.New()
	fmt.Println("GoRigumi initialized:", gorigumi)
}
```

---

### 2️⃣ Upload a File  

```go
func uploadHandler(w http.ResponseWriter, r *http.Request) {
	gorigumi := gorigumi.New()
	gorigumi.MaxFileSize = 10 * 1024 * 1024 // 10MB

	uploadedFile, err := gorigumi.UploadFile(r, "./uploads", true)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	fmt.Fprintf(w, "File uploaded successfully: %+v", uploadedFile)
}
```

---

### 3️⃣ Read JSON from Request  

```go
type User struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	gorigumi := gorigumi.New()
	gorigumi.MaxJSONSize = 2 * 1024 * 1024 // 2MB

	var user User
	err := gorigumi.JSONRead(w, r, &user)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	gorigumi.JSONWrite(w, http.StatusOK, map[string]string{"message": "JSON received!"})
}
```

---

### 4️⃣ Convert a String to a URL-Friendly Slug  

```go
gorigumi := gorigumi.New()
slug, err := gorigumi.ConvertToSlug("Hello, World! 123")
if err != nil {
	fmt.Println("Error:", err)
} else {
	fmt.Println("Slug:", slug) // Output: "hello-world-123"
}
```

---

### 5️⃣ Send JSON to a Remote Server  

```go
gorigumi := gorigumi.New()
data := map[string]string{"message": "Hello, Server!"}

res, statusCode, err := gorigumi.JSONPushToRemote("https://example.com/api", data)
if err != nil {
	fmt.Println("Error sending JSON:", err)
} else {
	fmt.Println("Response Status Code:", statusCode)
}
```

---

## Running Tests 🧪  

```sh
go test ./... -v
```

---

## License 📜  

This project is licensed under the [MIT License](LICENSE).

---

🔥 **Happy Coding with `gorigumi`!** 🚀