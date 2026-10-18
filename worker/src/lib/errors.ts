/** Structured application error: HTTP status, machine-readable code, safe message. */
export class AppError extends Error {
	constructor(
		public readonly status: number,
		public readonly code: string,
		message: string,
	) {
		super(message);
	}
}

export function newAppError(status: number, code: string, message: string): AppError {
	return new AppError(status, code, message);
}

// Mirrors internal/errors/errors.go: NotFound/Validation/etc use a fixed code
// and the passed string as the message.
export function notFound(message: string): AppError {
	return new AppError(404, "NOT_FOUND", message);
}
export function validation(message: string): AppError {
	return new AppError(400, "VALIDATION_ERROR", message);
}
export function unauthorized(message: string): AppError {
	return new AppError(401, "UNAUTHORIZED", message);
}
export function conflict(message: string): AppError {
	return new AppError(409, "CONFLICT", message);
}
export function internal(message: string): AppError {
	return new AppError(500, "INTERNAL_ERROR", message);
}