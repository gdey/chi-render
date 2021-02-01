//
// BLOG
// ====
// This example demonstrates a HTTP REST web service with some fixture data.
// Follow along the example and patterns.
//
// Also check routes.json for the generated docs from passing the -routes flag
//
// Boot the server:
// ----------------
// $ go run main.go
//
// Client requests:
// ----------------
// $ curl http://localhost:3333/
// root.
//
// $ curl http://localhost:3333/articles
// [{"id":"1","title":"Hi"},{"id":"2","title":"sup"}]
//
// $ curl http://localhost:3333/articles/1
// {"id":"1","title":"Hi"}
//
// $ curl -X DELETE http://localhost:3333/articles/1
// {"id":"1","title":"Hi"}
//
// $ curl http://localhost:3333/articles/1
// "Not Found"
//
// $ curl -X POST -d '{"id":"will-be-omitted","title":"awesomeness"}' http://localhost:3333/articles
// {"id":"97","title":"awesomeness"}
//
// $ curl http://localhost:3333/articles/97
// {"id":"97","title":"awesomeness"}
//
// $ curl http://localhost:3333/articles
// [{"id":"2","title":"sup"},{"id":"97","title":"awesomeness"}]
//
package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"strings"

	"github.com/gdey/chi-render/responders"

	render "github.com/gdey/chi-render"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/docgen"
)

var routes = flag.Bool("routes", false, "Generate router documentation")

func main() {
	flag.Parse()

	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	//	r.Use(render.SetContentType(render.ContentTypeJSON))

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("root."))
	})

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("pong"))
	})

	r.Get("/panic", func(w http.ResponseWriter, r *http.Request) {
		panic("test")
	})

	// RESTy routes for "articles" resource
	r.Route("/articles", func(r chi.Router) {
		ctrl := render.CloneDefault()
		_ = ctrl.SetResponder(render.ContentTypeHTML, responders.HTML)
		r.Use(render.WithCtx(ctrl))
		r.With(paginate).Get("/", ListArticles)
		r.Post("/", CreateArticle)       // POST /articles
		r.Get("/search", SearchArticles) // GET /articles/search

		r.Route("/{articleID}", func(r chi.Router) {
			r.Use(ArticleCtx)            // Load the *Article on the request context
			r.Get("/", GetArticle)       // GET /articles/123
			r.Put("/", UpdateArticle)    // PUT /articles/123
			r.Delete("/", DeleteArticle) // DELETE /articles/123
		})

		// GET /articles/whats-up
		r.With(ArticleCtx).Get("/{articleSlug:[a-z-]+}", GetArticle)
	})

	// Mount the admin sub-router, which btw is the same as:
	// r.Route("/admin", func(r chi.Router) { admin routes here })
	r.Mount("/admin", adminRouter())

	// Passing -routes to the program will generate docs for the above
	// router definition. See the `routes.json` file in this folder for
	// the output.
	if *routes {
		// fmt.Println(docgen.JSONRoutesDoc(r))
		fmt.Println(docgen.MarkdownRoutesDoc(r, docgen.MarkdownOpts{
			ProjectPath: "github.com/go-chi/chi",
			Intro:       "Welcome to the chi/_examples/rest generated docs.",
		}))
		return
	}

	_ = http.ListenAndServe(":3333", r)
}

func ListArticles(w http.ResponseWriter, r *http.Request) {
	if err := render.RenderList(w, r, NewArticleListResponse(articles)); err != nil {
		renderErr := &ErrRender{}
		renderErr.Err = err
		_ = render.Render(w, r, renderErr)
		return
	}
}

// ArticleCtx middleware is used to load an Article object from
// the URL parameters passed through as the request. In case
// the Article could not be found, we stop here and return a 404.
func ArticleCtx(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var article *Article
		var err error

		if articleID := chi.URLParam(r, "articleID"); articleID != "" {
			article, err = dbGetArticle(articleID)
		} else if articleSlug := chi.URLParam(r, "articleSlug"); articleSlug != "" {
			article, err = dbGetArticleBySlug(articleSlug)
		} else {
			_ = render.Render(w, r, &ErrNotFound{Resource: fmt.Sprintf("article %v", articleSlug)})
			return
		}
		if err != nil {
			_ = render.Render(w, r, &ErrNotFound{Resource: "article"})
			return
		}

		ctx := context.WithValue(r.Context(), "article", article)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SearchArticles searches the Articles data for a matching article.
// It's just a stub, but you get the idea.
func SearchArticles(w http.ResponseWriter, r *http.Request) {
	_ = render.FromContext(r).RenderList(w, r, NewArticleListResponse(articles))
}

// CreateArticle persists the posted Article and returns it
// back to the client as an acknowledgement.
func CreateArticle(w http.ResponseWriter, r *http.Request) {
	render := render.FromContext(r)
	data := &ArticleRequest{}
	if err := render.Bind(r, data); err != nil {
		invalidRequest := &ErrInvalidRequest{}
		invalidRequest.Err = err
		_ = render.Render(w, r, invalidRequest)
		return
	}

	article := data.Article
	_, _ = dbNewArticle(article)

	render.Status(r, http.StatusCreated)
	_ = render.Render(w, r, NewArticleResponse(article))
}

// GetArticle returns the specific Article. You'll notice it just
// fetches the Article right off the context, as its understood that
// if we made it this far, the Article must be on the context. In case
// its not due to a bug, then it will panic, and our Recoverer will save us.
func GetArticle(w http.ResponseWriter, r *http.Request) {
	render := render.FromContext(r)
	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the ArticleCtx
	// middleware. The worst case, the recoverer middleware will save us.
	article := r.Context().Value("article").(*Article)

	if err := render.Render(w, r, NewArticleResponse(article)); err != nil {
		renderErr := &ErrRender{}
		renderErr.Err = err
		_ = render.Render(w, r, renderErr)
		return
	}
}

// UpdateArticle updates an existing Article in our persistent store.
func UpdateArticle(w http.ResponseWriter, r *http.Request) {
	render := render.FromContext(r)
	article := r.Context().Value("article").(*Article)

	data := &ArticleRequest{Article: article}
	if err := render.Bind(r, data); err != nil {
		invalidRequest := &ErrInvalidRequest{}
		invalidRequest.Err = err
		_ = render.Render(w, r, invalidRequest)
		return
	}
	article = data.Article
	_, _ = dbUpdateArticle(article.ID, article)

	_ = render.Render(w, r, NewArticleResponse(article))
}

// DeleteArticle removes an existing Article from our persistent store.
func DeleteArticle(w http.ResponseWriter, r *http.Request) {
	var err error
	render := render.FromContext(r)

	// Assume if we've reach this far, we can access the article
	// context because this handler is a child of the ArticleCtx
	// middleware. The worst case, the recoverer middleware will save us.
	article := r.Context().Value("article").(*Article)

	article, err = dbRemoveArticle(article.ID)
	if err != nil {
		invalidRequest := &ErrInvalidRequest{}
		invalidRequest.Err = err
		_ = render.Render(w, r, invalidRequest)
		return
	}

	_ = render.Render(w, r, NewArticleResponse(article))
}

// A completely separate router for administrator routes
func adminRouter() chi.Router {
	r := chi.NewRouter()
	r.Use(AdminOnly)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("admin: index"))
	})
	r.Get("/accounts", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("admin: list accounts.."))
	})
	r.Get("/users/{userId}", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(fmt.Sprintf("admin: view user id %v", chi.URLParam(r, "userId"))))
	})
	return r
}

// AdminOnly middleware restricts access to just administrators.
func AdminOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAdmin, ok := r.Context().Value("acl.admin").(bool)
		if !ok || !isAdmin {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// paginate is a stub, but very possible to implement middleware logic
// to handle the request params for handling a paginated request.
func paginate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// just a stub.. some ideas are to look at URL query params for something like
		// the page number, or the limit, and send a query cursor down the chain
		next.ServeHTTP(w, r)
	})
}

//--
// Request and Response payloads for the REST api.
//
// The payloads embed the data model objects an
//
// In a real-world project, it would make sense to put these payloads
// in another file, or another sub-package.
//--

type UserPayload struct {
	*User
	Role string `json:"role"`

	render.NilBinder
}

func NewUserPayloadResponse(user *User) *UserPayload {
	return &UserPayload{User: user}
}

func (u *UserPayload) Render(_ http.ResponseWriter, _ *http.Request) error {
	u.Role = "collaborator"
	return nil
}

// ArticleRequest is the request payload for Article data model.
//
// NOTE: It's good practice to have well defined request and response payloads
// so you can manage the specific inputs and outputs for clients, and also gives
// you the opportunity to transform data on input or output, for example
// on request, we'd like to protect certain fields and on output perhaps
// we'd like to include a computed field based on other values that aren't
// in the data model. Also, check out this awesome blog post on struct composition:
// http://attilaolah.eu/2014/09/10/json-and-struct-composition-in-go/
type ArticleRequest struct {
	*Article

	User *UserPayload `json:"user,omitempty"`

	ProtectedID string `json:"id"` // override 'id' json to have more control
}

func (a *ArticleRequest) Bind(_ *http.Request) error {
	// a.Article is nil if no Article fields are sent in the request. Return an
	// error to avoid a nil pointer dereference.
	if a.Article == nil {
		return ErrArticleMissingField
	}

	// a.User is nil if the UserPayload has not been sent in the request. In this app
	// this won't cause a panic, but checks in the Bind method may be required if
	// a.User or further nested fields like a.User.Name are accessed elsewhere.

	// just a post-process after a decode..
	a.ProtectedID = ""                                 // unset the protected ID
	a.Article.Title = strings.ToLower(a.Article.Title) // as an example, we down-case
	return nil
}

// ArticleResponse is the response payload for the Article data model.
// See NOTE above in ArticleRequest as well.
//
// In the ArticleResponse object, first a Render() is called on itself,
// then the next field, and so on, all the way down the tree.
// Render is called in top-down order, like a http handler middleware chain.
type ArticleResponse struct {
	*Article

	User *UserPayload `json:"user,omitempty"`

	// We add an additional field to the response here.. such as this
	// elapsed computed property
	Elapsed int64 `json:"elapsed"`
}

func NewArticleResponse(article *Article) *ArticleResponse {
	resp := &ArticleResponse{Article: article}

	if resp.User == nil {
		if user, _ := dbGetUser(resp.UserID); user != nil {
			resp.User = NewUserPayloadResponse(user)
		}
	}

	return resp
}

func (rd *ArticleResponse) Render(_ http.ResponseWriter, _ *http.Request) error {
	// Pre-processing before a response is marshalled and sent across the wire
	rd.Elapsed = 10
	return nil
}

var articleTemplate = template.Must(template.New("article").Parse(`<!DOCTYPE html>
<html>
<head>
<title>{{ .Title }}</title>
</head>
<body>
<h1>{{ .Title }}</h1>
<div><a href="/articles/{{ .Slug }}">by {{ .User.Name }} took {{ .Elapsed }}s</a></div>
<div id="main_body">
Body of article
</div>
</body>
</html>
`))

func (rd *ArticleResponse) MarshalHTML() ([]byte, error) {
	var buff bytes.Buffer
	err := articleTemplate.Execute(&buff, rd)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func NewArticleListResponse(articles []*Article) []render.Renderer {
	list := make([]render.Renderer, 0, len(articles))
	for _, article := range articles {
		list = append(list, NewArticleResponse(article))
	}
	return list
}

// NOTE: as a thought, the request and response payloads for an Article could be the
// same payload type, perhaps will do an example with it as well.
// type ArticlePayload struct {
//   *Article
// }

//--
// Error response payloads & renderers
//--

// ErrResponse renderer type for handling all sorts of errors.
//
// In the best case scenario, the excellent github.com/pkg/errors package
// helps reveal information on the error, setting it on Err, and in the Render()
// method, using it to set the application-specific error code in AppCode.

type ErrInvalidRequest struct {
	render.ErrResponse
}

func (err *ErrInvalidRequest) Render(_ http.ResponseWriter, _ *http.Request) error {
	err.StatusCode = 400
	return nil
}

type ErrRender struct {
	render.ErrResponse
}

func (err *ErrRender) Render(_ http.ResponseWriter, _ *http.Request) error {
	err.StatusCode = 422
	err.StatusText = "Error rendering response."
	return nil
}

// ErrNotFound is returned when the requested resource is not available in the datastore
type ErrNotFound struct {
	render.ErrResponse
	// Resource is the name of the object/page that was not found
	Resource string
}

func (err *ErrNotFound) Render(_ http.ResponseWriter, _ *http.Request) error {
	err.StatusCode = 404
	if err.Resource != "" {
		err.ErrorText = fmt.Sprintf("%v resource not found", err.Resource)
	}
	return nil
}

type ErrString string

func (err ErrString) Error() string { return string(err) }

const (
	ErrArticleNotFound     ErrString = "article not found"
	ErrArticleMissingField ErrString = "missing required Article fields"
	ErrUserNotFound        ErrString = "user not found"
)

//--
// Data model objects and persistence mocks:
//--

// User data model
type User struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`

	render.NilRender
}

// Article data model. I suggest looking at https://upper.io for an easy
// and powerful data persistence adapter.
type Article struct {
	ID     string `json:"id"`
	UserID int64  `json:"user_id"` // the author
	Title  string `json:"title"`
	Slug   string `json:"slug"`
}

// Article fixture data
var articles = []*Article{
	{ID: "1", UserID: 100, Title: "Hi", Slug: "hi"},
	{ID: "2", UserID: 200, Title: "sup", Slug: "sup"},
	{ID: "3", UserID: 300, Title: "alo", Slug: "alo"},
	{ID: "4", UserID: 400, Title: "bonjour", Slug: "bonjour"},
	{ID: "5", UserID: 500, Title: "whats up", Slug: "whats-up"},
}

// User fixture data
var users = []*User{
	{ID: 100, Name: "Peter"},
	{ID: 200, Name: "Julia"},
}

func dbNewArticle(article *Article) (string, error) {
	article.ID = fmt.Sprintf("%d", rand.Intn(100)+10)
	articles = append(articles, article)
	return article.ID, nil
}

func dbGetArticle(id string) (*Article, error) {
	for _, a := range articles {
		if a.ID == id {
			return a, nil
		}
	}
	return nil, ErrArticleNotFound
}

func dbGetArticleBySlug(slug string) (*Article, error) {
	for _, a := range articles {
		if a.Slug == slug {
			return a, nil
		}
	}
	return nil, ErrArticleNotFound
}

func dbUpdateArticle(id string, article *Article) (*Article, error) {
	for i, a := range articles {
		if a.ID == id {
			articles[i] = article
			return article, nil
		}
	}
	return nil, ErrArticleNotFound
}

func dbRemoveArticle(id string) (*Article, error) {
	for i, a := range articles {
		if a.ID == id {
			articles = append((articles)[:i], (articles)[i+1:]...)
			return a, nil
		}
	}
	return nil, ErrArticleNotFound
}

func dbGetUser(id int64) (*User, error) {
	for _, u := range users {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, ErrUserNotFound
}
