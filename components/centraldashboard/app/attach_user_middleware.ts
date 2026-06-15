import {NextFunction, Request, RequestHandler, Response} from 'express';

/**
 * Parses a groups header value that may be plain text or a base64-encoded
 * comma-separated list. Auto-detects base64 via round-trip check.
 */
function parseGroupsHeader(value: string): string[] {
  let raw = value;
  try {
    const decoded = Buffer.from(value, 'base64').toString('utf-8');
    if (Buffer.from(decoded).toString('base64') === value) {
      raw = decoded;
    }
  } catch (_) { /* not valid base64, use raw value */ }
  return raw.split(',').map((g) => g.trim()).filter(Boolean);
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
