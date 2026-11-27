# polymath-go-quiz
constant leaner - quiz service

### Phase 1
 
- ~~add HTTP handlers that displays the first quiz question in a simple HTML page.~~
- ~~Once you can open it in a browser, add answer submission and question progression~~
- ~~Build a Quiz object containing the existing questions and set its starting status to the first question.~~
- ~~Render the current question prompt and options as simple HTML.~~ (radio buttons deferred to Phase 2)
- ~~Add an HTML form whose submit button sends the answer.~~ (submits to `POST /` instead of `POST /answer`; the route name is deferred to Phase 2)
- ~~Add a handler that reads the submitted answer, updates Spec.UserAnswer, and increments SubmissionId.~~
- ~~Reuse isAnswerCorrect and update Status.Phase and Status.Feedback in the answer handler.~~
- ~~Increment Status.CurrentQuestion and either show the next question or set Status.Phase to QuizPhaseDone.~~
- ~~Remove the CLI input loop and time.Sleep from main.~~
- ~~Initialize the quiz once and start http.ListenAndServe on a port such as 8080.~~
- ~~Run go run ., open http://localhost:8083, and verify the full quiz flow in a browser.~~
- ~~Add focused httptest cases for the initial page, correct answer, incorrect answer, and finished state.~~

Added beyond the original plan:

- ~~Show the last answer's feedback and the final `Phase: Done` on the completion page, each message in its own paragraph.~~
- ~~Use Post/Redirect/Get: `POST /` records the answer and redirects (303) to `GET /`, so a refresh no longer resubmits the form or skips a question.~~
- ~~Keep the previous answer's feedback in `Status.LastResult` so it survives the redirect.~~
- ~~Add `POST /restart` and a "Restart quiz" button on the completion page.~~
- ~~Guard the shared quiz state with a mutex.~~
- ~~Share one `*Quiz` (with its mutex) across handlers instead of copying it, so every handler sees the same submission and restart state.~~
- ~~Protect `POST /` and `POST /restart` with a CSRF token: a random token is generated at startup, embedded in both forms as a hidden field, and checked in constant time (403 if missing or wrong).~~
- ~~Validate that the submitted answer is one of the current question's options (case-insensitive). An invalid answer returns 400 and re-renders the question with "Please choose one of the listed options." without advancing the quiz.~~
- The `Pending` phase is deliberately not displayed.
- Known limitations (addressed in Phase 2): one global quiz is shared by every visitor, and the CSRF token is one per server process, not tied to a session.

### Phase 2

- Spec scope: describe a new JSON API (`GET /quiz`, `POST /answers`, `POST /restart`) that the HTML page also uses, rather than the existing HTML form routes. It fits oapi-codegen far better.
- HTML page: keep it, served by the same server and calling the same quiz logic.
- Quiz ownership: add a quiz/session ID instead of one global quiz shared by every visitor. It changes every endpoint's shape, so decide it before writing the spec.
- Server style: oapi-codegen strict-server on plain net/http, which gives typed responses, including redirects.
- Layout: cmd/quiz-server for startup, internal/quiz for the logic, internal/web for the handlers and embedded HTML, internal/api for generated code.
- Generated files: commit the OpenAPI spec and the generated Go code. One `go:generate` or Makefile target runs TypeSpec and then oapi-codegen, so a plain `go build` works without Node.
- Phase 1 leftovers: the radio-button options and `POST /answer` are deliberately deferred. The JSON API's `POST /answers` replaces the old route name.

#### Tasks

1. ~~Move startup into cmd/quiz-server and the quiz logic, HTTP handling, and embedded HTML into the internal packages above.~~
2. ~~Add the quiz/session ID to the quiz state and handlers.~~
3. ~~Define the quiz API in TypeSpec.~~
4. ~~Generate the OpenAPI specification from TypeSpec.~~
5. ~~Use oapi-codegen to generate the Go server types and strict-server interface from OpenAPI.~~
6. Integrate the quiz logic with the generated server, port the existing tests, and verify the endpoints with tests.
7. Point the HTML page at the JSON API and re-check the full flow in a browser.
