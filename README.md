# polymath-go-quiz
constant leaner - quiz service

### Planning
 
- add HTTP handlers that displays the first quiz question in a simple HTML page.
- Once you can open it in a browser, add answer submission and question progression
- Build a Quiz object containing the existing questions and set its starting status to the first question.
- Render the current question prompt and radio-button options as simple HTML.
- Add an HTML form whose submit button sends the selected answer to POST /answer.
- Add a handler that reads the submitted answer, updates Spec.UserAnswer, and increments SubmissionId.
- Reuse isAnswerCorrect and update Status.Phase and Status.Feedback in the answer handler.
- Increment Status.CurrentQuestion and either show the next question or set Status.Phase to QuizPhaseDone.
- Remove the CLI input loop and time.Sleep from main.
- Initialize the quiz once and start http.ListenAndServe on a port such as 8080.
- Run go run ., open http://localhost:8083, and verify the full quiz flow in a browser.
- Add focused httptest cases for the initial page, correct answer, incorrect answer, and finished state.
