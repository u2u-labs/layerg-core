import {HttpEvent, HttpHandler, HttpInterceptor, HttpRequest} from '@angular/common/http';
import {AuthenticationService} from './authentication.service';
import {Observable} from 'rxjs';
import {Injectable} from '@angular/core';

@Injectable()
export class SessionInterceptor implements HttpInterceptor {
  constructor(private readonly authenticationService: AuthenticationService) {}

  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    const session = this.authenticationService.currentSessionValue;
    if (session && session.token) {
      req = req.clone({
        setHeaders: {
          Authorization: `Bearer ${session.token}`
        }
      });
    }
    return next.handle(req);
  }
}
