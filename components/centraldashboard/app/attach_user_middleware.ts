import {NextFunction, Request, RequestHandler, Response} from 'express';

function parseGroupsHeader(value: string): string[] {
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
