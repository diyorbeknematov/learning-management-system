// Command seed fills a running system with demo data, so the screens can be
// tried with something on them: categories, instructors, students, published
// courses with lessons and quizzes, enrollments, progress, certificates and
// reviews.
//
// It only uses the public HTTP API, as a SuperAdmin, instructors and students
// would, so it also exercises the API. It can be run again: what exists is
// left as it is.
//
//	make seed
//
// Settings (environment): SEED_API (default http://localhost:8080),
// ADMIN_USERNAME (default admin) and ADMIN_PASSWORD (the first SuperAdmin).
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

const demoPassword = "Demo-Passw0rd!2024"

// --- the demo data

type option struct {
	text    string
	correct bool
}

type question struct {
	text    string
	kind    string
	options []option
}

type lesson struct {
	title    string
	minutes  int
	preview  bool
	material string
}

type module struct {
	title   string
	lessons []lesson
}

type course struct {
	title       string
	instructor  string
	category    string
	difficulty  string
	price       float64
	payoutShare float64 // percent of the price for the instructor; 0 is none
	language    string
	minutes     int
	description string
	outcomes    []string
	requires    []string
	modules     []module
	quiz        []question
	draft       bool
}

type person struct {
	username, first, last, role string
}

var categories = []string{"Programming", "Design", "Business"}

var people = []person{
	{"ali.teacher", "Ali", "Karimov", "Instructor"},
	{"sara.teacher", "Sara", "Nazarova", "Instructor"},
	{"student1", "Bobur", "Aliyev", "Student"},
	{"student2", "Dilnoza", "Rahimova", "Student"},
	{"student3", "Jasur", "Tursunov", "Student"},
	{"student4", "Malika", "Yusupova", "Student"},
	{"student5", "Otabek", "Saidov", "Student"},
}

func yesNo(text string, yes bool) question {
	return question{text: text, kind: "true_false", options: []option{{"True", yes}, {"False", !yes}}}
}

var courses = []course{
	{
		title: "Go for Beginners", instructor: "ali.teacher", category: "Programming", difficulty: "beginner",
		price: 0, language: "English", minutes: 120,
		description: "Start with Go: the tools, the syntax and your first programs.",
		outcomes:    []string{"Write and run Go programs", "Use slices, maps and structs", "Handle errors the Go way"},
		requires:    []string{"A computer", "No programming experience is needed"},
		modules: []module{
			{"Getting started", []lesson{
				{"Why Go?", 8, true, "Go is a small, fast, compiled language made for servers and tools."},
				{"Installing Go", 10, true, "Download Go from go.dev, then run `go version` to check it."},
				{"Hello, world", 12, false, "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"Hello, world\")\n}"},
			}},
			{"The language", []lesson{
				{"Variables and types", 15, false, "Use := inside functions and var at the top level."},
				{"Slices and maps", 20, false, "A slice is a view of an array; a map is a hash table."},
				{"Errors", 15, false, "A function returns an error as its last value; check it with if err != nil."},
			}},
		},
		quiz: []question{
			{"Which keyword starts a function?", "single_choice", []option{{"func", true}, {"def", false}, {"function", false}}},
			{"Which of these are Go types?", "multiple_choice", []option{{"int", true}, {"string", true}, {"var", false}}},
			yesNo("Go is a compiled language.", true),
		},
	},
	{
		title: "Concurrency in Go", instructor: "ali.teacher", category: "Programming", difficulty: "advanced",
		price: 49, payoutShare: 30, language: "English", minutes: 180,
		description: "Goroutines, channels and the patterns that keep concurrent code simple.",
		outcomes:    []string{"Start and stop goroutines safely", "Pipelines with channels", "Avoid races and leaks"},
		requires:    []string{"Go basics"},
		modules: []module{
			{"Goroutines", []lesson{
				{"The go keyword", 12, true, "go f() runs f in a new goroutine."},
				{"WaitGroup", 14, false, "A sync.WaitGroup waits for goroutines to finish."},
			}},
			{"Channels", []lesson{
				{"Sending and receiving", 15, false, "ch <- v sends, <-ch receives."},
				{"select", 18, false, "select waits on several channel operations at once."},
			}},
		},
		quiz: []question{
			{"What starts a goroutine?", "single_choice", []option{{"go", true}, {"async", false}, {"spawn", false}}},
			yesNo("Reading from a nil channel blocks forever.", true),
		},
	},
	{
		title: "UI Design Fundamentals", instructor: "sara.teacher", category: "Design", difficulty: "beginner",
		price: 29, payoutShare: 40, language: "English", minutes: 150,
		description: "Layout, colour and type: the rules that make an interface easy to use.",
		outcomes:    []string{"Build a clear visual hierarchy", "Choose colours that are accessible", "Design for small screens"},
		requires:    []string{"No tools needed to start"},
		modules: []module{
			{"Basics", []lesson{
				{"Layout and spacing", 14, true, "Use a grid and keep spacing consistent."},
				{"Colour and contrast", 16, false, "Text needs a contrast ratio of at least 4.5 to 1."},
			}},
			{"Practice", []lesson{
				{"Typography", 12, false, "Limit yourself to two typefaces."},
				{"A first screen", 25, false, "Sketch the screen on paper, then build it."},
			}},
		},
		quiz: []question{
			{"Minimum contrast ratio for body text?", "single_choice", []option{{"4.5 : 1", true}, {"2 : 1", false}, {"1 : 1", false}}},
			yesNo("Use as many typefaces as possible.", false),
		},
	},
	{
		title: "Marketing Basics", instructor: "sara.teacher", category: "Business", difficulty: "intermediate",
		price: 19, payoutShare: 25, language: "English", minutes: 90,
		description: "Know your customer, pick a channel, measure the result.",
		outcomes:    []string{"Describe your customer", "Choose a channel", "Read the numbers"},
		requires:    []string{"A product or an idea"},
		modules: []module{
			{"Customers", []lesson{
				{"Who is your customer?", 10, true, "Write one paragraph about a real person."},
				{"Where do they look?", 12, false, "Go where your customers already are."},
			}},
		},
		quiz: []question{yesNo("You should measure the result of a campaign.", true)},
	},
	{
		title: "Draft: Data Structures", instructor: "ali.teacher", category: "Programming", difficulty: "intermediate",
		price: 15, payoutShare: 30, language: "English", minutes: 60,
		description: "A course that is still being written (a draft, visible only to its owner).",
		outcomes:    []string{"Choose the right data structure"},
		modules:     []module{{"Lists", []lesson{{"Arrays and lists", 10, false, "Coming soon."}}}},
		draft:       true,
	},
}

// what the students do: how far each goes in a course
type activity struct {
	student string
	course  string
	mode    string // "start", "half", "finish" (all lessons and the quiz), "fail" (all lessons, quiz answered wrong)
	review  int    // stars, 0 for none
	comment string
}

var activities = []activity{
	{"student1", "Go for Beginners", "finish", 5, "Clear and practical. Loved the examples."},
	{"student2", "Go for Beginners", "finish", 4, "Good pace for a beginner."},
	{"student3", "Go for Beginners", "half", 0, ""},
	{"student4", "Go for Beginners", "start", 0, ""},
	{"student1", "Concurrency in Go", "half", 0, ""},
	{"student2", "Concurrency in Go", "finish", 5, "Channels finally make sense."},
	{"student5", "Concurrency in Go", "fail", 0, ""},
	{"student3", "UI Design Fundamentals", "finish", 4, "Short and to the point."},
	{"student4", "UI Design Fundamentals", "half", 0, ""},
	{"student5", "UI Design Fundamentals", "finish", 3, "Useful, I wanted more practice."},
	{"student1", "Marketing Basics", "start", 0, ""},
	{"student4", "Marketing Basics", "finish", 5, "Simple and honest."},
}

// --- a small client of the API

type client struct {
	base string
}

type result struct {
	status int
	data   json.RawMessage
	err    string
}

func (c client) do(token, method, path string, body any) result {
	for attempt := 0; ; attempt++ {
		var reader io.Reader

		if body != nil {
			raw, err := json.Marshal(body)
			if err != nil {
				log.Fatal(err)
			}

			reader = bytes.NewReader(raw)
		}

		req, err := http.NewRequest(method, c.base+"/api/v1"+path, reader)
		if err != nil {
			log.Fatal(err)
		}

		req.Header.Set("Content-Type", "application/json")

		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatalf("the API is not reachable at %s: %v (is the system running?)", c.base, err)
		}

		raw, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		// a rate limit: wait as long as the server says, then try again
		if resp.StatusCode == http.StatusTooManyRequests && attempt < 5 {
			wait, _ := strconv.Atoi(resp.Header.Get("Retry-After"))
			if wait < 1 {
				wait = 5
			}

			log.Printf("rate limit, waiting %d s", wait)
			time.Sleep(time.Duration(wait) * time.Second)

			continue
		}

		var envelope struct {
			Data  json.RawMessage `json:"data"`
			Error struct {
				Message string            `json:"message"`
				Fields  map[string]string `json:"fields"`
			} `json:"error"`
		}

		_ = json.Unmarshal(raw, &envelope)

		message := envelope.Error.Message
		if len(envelope.Error.Fields) > 0 {
			message += fmt.Sprintf(" %v", envelope.Error.Fields)
		}

		return result{status: resp.StatusCode, data: envelope.Data, err: message}
	}
}

// must calls the API and stops the program when it fails.
func (c client) must(token, method, path string, body any, out any) {
	r := c.do(token, method, path, body)
	if r.status >= 300 {
		log.Fatalf("%s %s: %d %s", method, path, r.status, r.err)
	}

	if out != nil && len(r.data) > 0 {
		if err := json.Unmarshal(r.data, out); err != nil {
			log.Fatalf("%s %s: %v", method, path, err)
		}
	}
}

func (c client) login(username, password string) string {
	var out struct {
		AccessToken string `json:"access_token"`
	}

	c.must("", http.MethodPost, "/auth/login", map[string]string{"username": username, "password": password}, &out)

	return out.AccessToken
}

// --- the steps

type idName struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Username string `json:"username"`
	Title    string `json:"title"`
}

type page struct {
	Items []idName `json:"items"`
}

func ensureCategories(c client, admin string) map[string]string {
	var existing page

	c.must(admin, http.MethodGet, "/categories?limit=100", nil, &existing)

	ids := map[string]string{}
	for _, category := range existing.Items {
		ids[category.Name] = category.ID
	}

	for _, name := range categories {
		if _, ok := ids[name]; ok {
			continue
		}

		var created idName

		c.must(admin, http.MethodPost, "/categories", map[string]string{"name": name}, &created)
		ids[name] = created.ID

		log.Printf("category %q", name)
	}

	return ids
}

func ensurePeople(c client, admin string) map[string]string {
	ids := map[string]string{}

	for _, p := range people {
		var found page

		c.must(admin, http.MethodGet, "/users?limit=20&search="+url.QueryEscape(p.username), nil, &found)

		for _, user := range found.Items {
			if user.Username == p.username {
				ids[p.username] = user.ID
			}
		}

		if ids[p.username] != "" {
			continue
		}

		var created idName

		c.must(admin, http.MethodPost, "/users", map[string]string{
			"first_name": p.first, "last_name": p.last, "username": p.username,
			"email": p.username + "@demo.example.com", "password": demoPassword, "role": p.role,
		}, &created)
		ids[p.username] = created.ID

		log.Printf("%s %s", strings.ToLower(p.role), p.username)
	}

	return ids
}

type courseDetail struct {
	ID      string `json:"id"`
	Title   string `json:"title"`
	Modules []struct {
		Lessons []struct {
			ID string `json:"id"`
		} `json:"lessons"`
	} `json:"modules"`
}

func (d courseDetail) lessonIDs() []string {
	var ids []string

	for _, m := range d.Modules {
		for _, l := range m.Lessons {
			ids = append(ids, l.ID)
		}
	}

	return ids
}

// createCourse builds one course as its instructor, and returns false when it was there already.
func createCourse(c client, admin, token string, spec course, categoryID string, instructorID string) (string, bool) {
	var found page

	c.must(token, http.MethodGet, "/courses?limit=20&instructor_id="+instructorID+"&q="+url.QueryEscape(spec.title), nil, &found)

	for _, existing := range found.Items {
		if existing.Title == spec.title {
			return existing.ID, false
		}
	}

	var created courseDetail

	c.must(token, http.MethodPost, "/courses", map[string]any{
		"title": spec.title, "description": spec.description, "category_id": categoryID, "difficulty": spec.difficulty,
		"language": spec.language, "price": spec.price, "total_duration": spec.minutes,
		"learning_outcomes": spec.outcomes, "requirements": spec.requires,
	}, &created)

	if spec.payoutShare > 0 {
		// only a SuperAdmin sets the payout
		c.must(admin, http.MethodPut, "/courses/"+created.ID, map[string]any{"payout_type": "percentage", "payout_value": spec.payoutShare}, nil)
	}

	var lessonIDs []string

	for _, m := range spec.modules {
		var mod idName

		c.must(token, http.MethodPost, "/courses/"+created.ID+"/modules", map[string]string{"title": m.title}, &mod)

		for _, l := range m.lessons {
			var made idName

			c.must(token, http.MethodPost, "/modules/"+mod.ID+"/lessons", map[string]any{"title": l.title, "duration": l.minutes, "is_preview": l.preview}, &made)
			c.must(token, http.MethodPost, "/lessons/"+made.ID+"/materials", map[string]string{"type": "text", "content": l.material}, nil)

			lessonIDs = append(lessonIDs, made.ID)
		}
	}

	if len(spec.quiz) > 0 {
		var quiz idName

		c.must(token, http.MethodPost, "/courses/"+created.ID+"/quizzes", map[string]any{
			"title": "Final test", "description": "Check what you learned.", "time_limit": 20, "pass_threshold": 60, "max_attempts": 3,
		}, &quiz)

		for _, q := range spec.quiz {
			options := make([]map[string]any, len(q.options))
			for i, o := range q.options {
				options[i] = map[string]any{"option_text": o.text, "is_correct": o.correct}
			}

			c.must(token, http.MethodPost, "/quizzes/"+quiz.ID+"/questions", map[string]any{"text": q.text, "type": q.kind, "options": options}, nil)
		}
	}

	if !spec.draft {
		c.must(token, http.MethodPatch, "/courses/"+created.ID+"/status", map[string]string{"status": "published"}, nil)
	}

	log.Printf("course %q (%d lessons)", spec.title, len(lessonIDs))

	return created.ID, true
}

type quizQuestions []struct {
	ID      string `json:"id"`
	Options []struct {
		ID      string `json:"id"`
		Correct bool   `json:"is_correct"`
	} `json:"options"`
}

// study plays what one student does in one course.
func study(c client, teacherToken, studentToken string, a activity, courseID string) {
	if r := c.do(studentToken, http.MethodPost, "/courses/"+courseID+"/enrollments", nil); r.status >= 300 && r.status != http.StatusConflict {
		log.Fatalf("enroll %s in %q: %d %s", a.student, a.course, r.status, r.err)
	}

	var detail courseDetail

	c.must(studentToken, http.MethodGet, "/courses/"+courseID, nil, &detail)

	lessons := detail.lessonIDs()
	done := len(lessons)

	switch a.mode {
	case "start":
		done = 0
	case "half":
		done = (len(lessons) + 1) / 2
	}

	for _, id := range lessons[:done] {
		c.must(studentToken, http.MethodPost, "/lessons/"+id+"/progress", map[string]bool{"completed": true}, nil)
	}

	if a.mode == "finish" || a.mode == "fail" {
		takeQuiz(c, teacherToken, studentToken, courseID, a.mode == "finish")
	}

	if a.review > 0 {
		if r := c.do(studentToken, http.MethodPost, "/courses/"+courseID+"/reviews", map[string]any{"rating": a.review, "comment": a.comment}); r.status >= 300 && r.status != http.StatusConflict {
			log.Fatalf("review: %d %s", r.status, r.err)
		}
	}

	log.Printf("%s: %s (%s)", a.student, a.course, a.mode)
}

func takeQuiz(c client, teacherToken, studentToken, courseID string, pass bool) {
	var quizzes []idName

	c.must(studentToken, http.MethodGet, "/courses/"+courseID+"/quizzes", nil, &quizzes)

	if len(quizzes) == 0 {
		return
	}

	var attempts []struct {
		Passed      bool    `json:"passed"`
		CompletedAt *string `json:"completed_at"`
	}

	c.must(studentToken, http.MethodGet, "/quizzes/"+quizzes[0].ID+"/attempts", nil, &attempts)

	for _, attempt := range attempts {
		if attempt.Passed || (!pass && attempt.CompletedAt != nil) {
			return // done before
		}
	}

	// the instructor knows the right answers; the student only gets the questions
	var right quizQuestions

	c.must(teacherToken, http.MethodGet, "/quizzes/"+quizzes[0].ID+"/questions", nil, &right)

	var attempt struct {
		ID string `json:"id"`
	}

	c.must(studentToken, http.MethodPost, "/quizzes/"+quizzes[0].ID+"/attempts", nil, &attempt)

	answers := make([]map[string]any, 0, len(right))

	for _, q := range right {
		chosen := []string{}

		for _, o := range q.Options {
			if o.Correct == pass {
				chosen = append(chosen, o.ID)
			}
		}

		if !pass && len(chosen) > 1 {
			chosen = chosen[:1]
		}

		answers = append(answers, map[string]any{"question_id": q.ID, "option_ids": chosen})
	}

	c.must(studentToken, http.MethodPost, "/attempts/"+attempt.ID+"/submit", map[string]any{"answers": answers}, nil)
}

func main() {
	base := strings.TrimRight(getenv("SEED_API", "http://localhost:8080"), "/")
	adminUser := getenv("ADMIN_USERNAME", "admin")
	adminPassword := os.Getenv("ADMIN_PASSWORD")

	if adminPassword == "" {
		log.Fatal("ADMIN_PASSWORD is not set: it is the password of the first SuperAdmin")
	}

	c := client{base: base}
	admin := c.login(adminUser, adminPassword)

	categoryIDs := ensureCategories(c, admin)
	userIDs := ensurePeople(c, admin)

	tokens := map[string]string{}
	for _, p := range people {
		tokens[p.username] = c.login(p.username, demoPassword)
	}

	courseIDs := map[string]string{}

	for _, spec := range courses {
		id, _ := createCourse(c, admin, tokens[spec.instructor], spec, categoryIDs[spec.category], userIDs[spec.instructor])
		courseIDs[spec.title] = id
	}

	teacherOf := map[string]string{}
	for _, spec := range courses {
		teacherOf[spec.title] = tokens[spec.instructor]
	}

	for _, a := range activities {
		study(c, teacherOf[a.course], tokens[a.student], a, courseIDs[a.course])
	}

	fmt.Printf(`
Demo data is ready. Everybody has the password %s

  SuperAdmin   %s  (your ADMIN_PASSWORD)
  Instructors  ali.teacher, sara.teacher
  Students     student1 ... student5

`, demoPassword, adminUser)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
