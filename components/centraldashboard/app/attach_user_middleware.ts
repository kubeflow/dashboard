import {NextFunction, Request, RequestHandler, Response} from 'express';

/**
 * Parses a JSON array of strings out of `value`, returning undefined if
 * `value` is not valid JSON or is not an array of strings.
 */
function tryParseJsonStringArray(value: string): string[] | undefined {
  try {
    const parsed = JSON.parse(value);
    if (Array.isArray(parsed) && parsed.every((g) => typeof g === 'string')) {
      return parsed;
    }
  } catch {
    // Not valid JSON; fall through.
  }
  return undefined;
}

function parseGroupsHeader(value: string): string[] {
  // Istio's RequestAuthentication.outputClaimToHeaders base64-encodes
  // non-string JWT claims (e.g. an array `groups` claim) before injecting
  // them as a header, so this is the value's most common on-the-wire form.
  // Only trust the decoded value if it is unambiguously a JSON array of
  // strings, so a group literally named e.g. "dGVhbQ==" isn't silently
  // reinterpreted as a different identity.
  try {
    const decoded = Buffer.from(value, 'base64').toString('utf-8');
    const parsed = tryParseJsonStringArray(decoded);
    if (parsed) {
      return parsed;
    }
  } catch {
    // Not valid base64; fall through.
  }

  const parsed = tryParseJsonStringArray(value);
  if (parsed) {
    return parsed;
  }

  return value.split(',').map((g) => g.trim()).filter(Boolean);
}

/**
 * Returns a function that uses the provided header and prefix to extract
 * a User object with the requesting user's identity.
 */
export function attachUser(
    userIdHeader: string, userIdPrefix: string, groupsHeader: string): RequestHandler {
  return (req: Request, _: Response, next: NextFunction) => {
    let email = 'anonymous@kubeflow.org';
    let auth: User.AuthObject;
    let groups: string[] = [];
    if (userIdHeader && req.header(userIdHeader)) {
      email = req.header(userIdHeader).slice(userIdPrefix.length);
      auth = {[userIdHeader]: req.header(userIdHeader)};
    }

    if (groupsHeader && req.header(groupsHeader)) {
      groups = parseGroupsHeader(req.header(groupsHeader));
    }
    req.user = {
      email,
      username: email.split('@')[0],
      domain: email.split('@')[1],
      hasAuth: auth !== undefined,
      auth,
      groups,
    };
    next();
  };
}
