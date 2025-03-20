// import {HttpEvent, HttpHandler, HttpInterceptor, HttpRequest} from '@angular/common/http';
// import {AuthenticationService} from './authentication.service';
// import {Observable} from 'rxjs';
// import {Injectable} from '@angular/core';
//
// @Injectable()
// export class SessionInterceptor implements HttpInterceptor {
//   constructor(private readonly authenticationService: AuthenticationService) {}
//
//   intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
//     const session = this.authenticationService.currentSessionValue;
//     if (session && session.token) {
//       req = req.clone({
//         setHeaders: {
//           Authorization: `Bearer ${session.token}`
//         }
//       });
//     }
//     return next.handle(req);
//   }
// }

import { HttpEvent, HttpHandler, HttpInterceptor, HttpRequest } from '@angular/common/http';
import { AuthenticationService } from './authentication.service';
import { Observable } from 'rxjs';
import { Injectable } from '@angular/core';
import {environment, environmentLayerg} from '../environments/environment';

@Injectable()
export class SessionInterceptor implements HttpInterceptor {
  private coreApiUrl = environment.production ? document.location.origin : environment.apiBaseUrl; // API Layerg Core
  private hubApiUrl = environmentLayerg.production ? document.location.origin : environmentLayerg.apiBaseUrl; // API Layerg Hub

  constructor(private readonly authenticationService: AuthenticationService) {}

  intercept(req: HttpRequest<any>, next: HttpHandler): Observable<HttpEvent<any>> {
    let modifiedReq = req;

    // Lấy token của Layerg Core từ AuthenticationService
    const session = this.authenticationService.currentSessionValue;
    const coreToken = session && session.token ? session.token : '';

    // Lấy token của Layerg Hub từ localStorage hoặc nơi khác
    // const hubToken = localStorage.getItem('hub_token') || '';
    const hubToken = JSON.parse(localStorage.getItem('accessToken') as string);

    // Kiểm tra API nào đang được gọi để set token phù hợp
    if (req.url.startsWith(this.coreApiUrl) && coreToken) {
      modifiedReq = req.clone({
        setHeaders: {
          Authorization: `Bearer ${coreToken}`
        }
      });
    } else if (req.url.startsWith(this.hubApiUrl) && hubToken) {
      modifiedReq = req.clone({
        setHeaders: {
          Authorization: `Bearer ${hubToken?.accessToken}`
        }
      });
    }

    return next.handle(modifiedReq);
  }
}

