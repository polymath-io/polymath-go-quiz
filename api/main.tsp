// Based on the TypeSpec REST guide: https://typespec.io/docs/getting-started/getting-started-rest/01-setup-basic-syntax/
// Decorators: https://typespec.io/docs/libraries/http/reference/ (@route, @path, @statusCode, ...) and
// https://typespec.io/docs/libraries/openapi/reference/ (@operationId, @info).
import "@typespec/http";
import "@typespec/openapi";

using TypeSpec.Http;
using TypeSpec.OpenAPI;

/**
 * Quiz API. Each visitor plays their own quiz, identified by a session ID in
 * the path. The ID is a random 128-bit value and acts as the access key, so
 * there is no cookie or CSRF token: a cross-site request cannot target a quiz
 * without already knowing its ID.
 */
@service(#{ title: "Quiz API" })
@info(#{ version: "1.0.0" })
@server("http://localhost:8083", "Local development server")
namespace QuizApi;

/** Whether the quiz is still asking questions. */
enum QuizPhase {
  InProgress,
  Done,
}

/** A question as shown to the player. The correct answer is never exposed. */
model Question {
  prompt: string;

  /** Answers the player may choose from. */
  options: string[];
}

/** The outcome of the most recently submitted answer. */
model LastAnswer {
  correct: boolean;

  /** Human-readable feedback, such as "well done". */
  message: string;
}

/** Everything a client needs to render the quiz. */
model QuizState {
  /** Session ID of this quiz (UUID v4). */
  id: string;

  phase: QuizPhase;

  /** The question to answer now. Absent once the quiz is done. */
  question?: Question;

  /** 1-based number of the current question; equals totalQuestions once done. */
  questionNumber: int32;

  totalQuestions: int32;

  /** Absent until the first answer has been submitted. */
  lastAnswer?: LastAnswer;
}

model SubmitAnswerRequest {
  /** One of the current question's options, compared case-insensitively. */
  answer: string;
}

model ErrorBody {
  message: string;
}

model BadRequestError {
  @statusCode _: 400;
  @body body: ErrorBody;
}

model NotFoundError {
  @statusCode _: 404;
  @body body: ErrorBody;
}

model ConflictError {
  @statusCode _: 409;
  @body body: ErrorBody;
}

model CreatedQuiz {
  @statusCode _: 201;

  /** Path of the new quiz, /quizzes/{quizId}. */
  @header location: string;

  @body body: QuizState;
}

model QuizPath {
  /** Session ID returned when the quiz was created. */
  @path quizId: string;
}

@route("/quizzes")
interface Quizzes {
  /** Start a new quiz with its own state. */
  @post
  @operationId("createQuiz")
  create(): CreatedQuiz;

  /** Get the current state of a quiz. */
  @get
  @route("{quizId}")
  @operationId("getQuiz")
  get(...QuizPath): QuizState | NotFoundError;

  /**
   * Submit an answer to the current question. The response is the quiz state
   * after the answer, including `lastAnswer`.
   */
  @post
  @route("{quizId}/answers")
  @operationId("submitAnswer")
  submitAnswer(...QuizPath, @body request: SubmitAnswerRequest):
    | QuizState
    | BadRequestError
    | NotFoundError
    | ConflictError;

  /** Reset the quiz to its first question and clear the previous result. */
  @post
  @route("{quizId}/restart")
  @operationId("restartQuiz")
  restart(...QuizPath): QuizState | NotFoundError;
}
