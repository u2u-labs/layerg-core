import {Inject, Injectable} from '@angular/core';
import {HttpClient} from '@angular/common/http';
import {BehaviorSubject, EMPTY, Observable} from 'rxjs';
import {tap} from 'rxjs/operators';
import {ConsoleService, ConsoleSession, UserRole} from './console.service';
import {WINDOW} from './window.provider';
import {SegmentService} from 'ngx-segment-analytics';
import {environment} from "../environments/environment";

const SESSION_LOCALSTORAGE_KEY = 'currentSession';

@Injectable({
  providedIn: 'root'
})
export class AuthenticationService {
  private readonly currentSessionSubject: BehaviorSubject<ConsoleSession>;
  readonly currentSession: Observable<ConsoleSession>;

  constructor(
    @Inject(WINDOW) private window: Window,
    private segment: SegmentService,
    private readonly http: HttpClient,
    private readonly consoleService: ConsoleService
  ) {
    const restoredSession: ConsoleSession = JSON.parse(localStorage.getItem(SESSION_LOCALSTORAGE_KEY) as string);
    if (restoredSession && !environment.nt) {
      this.segmentIdentify(restoredSession);
    }
    this.currentSessionSubject = new BehaviorSubject<ConsoleSession>(restoredSession);
    this.currentSession = this.currentSessionSubject.asObservable();
  }

  public get currentSessionValue(): ConsoleSession {
    return this.currentSessionSubject.getValue();
  }

  public get username(): string {
    const token = this.currentSessionSubject.getValue().token;
    const claims = JSON.parse(atob(token.split('.')[1]));
    return claims.usn;
  }

  public get sessionRole(): UserRole {
    const token = this.currentSessionSubject.getValue().token;
    const claims = JSON.parse(atob(token.split('.')[1]));
    const role = claims.rol as number;
    switch (role) {
      case 1:
        return UserRole.USER_ROLE_ADMIN;
      case 2:
        return UserRole.USER_ROLE_DEVELOPER;
      case 3:
        return UserRole.USER_ROLE_MAINTAINER;
      case 4:
        return UserRole.USER_ROLE_READONLY;
      default:
        return UserRole.USER_ROLE_UNKNOWN;
    }
  }

  login(username: string, password: string): Observable<ConsoleSession> {
    return this.consoleService.authenticate({username, password}).pipe(tap(session => {
      localStorage.setItem(SESSION_LOCALSTORAGE_KEY, JSON.stringify(session));
      this.currentSessionSubject.next(session);
      if (!environment.nt) {
        this.segmentIdentify(session);
      }
    }));
  }

  logout(): Observable<any> {
    if (!this.currentSessionSubject.getValue()) {
      return EMPTY;
    }
    return this.consoleService.authenticateLogout('', {
      token: this.currentSessionSubject.getValue()?.token,
    }).pipe(tap(() => {
      localStorage.removeItem(SESSION_LOCALSTORAGE_KEY);
      this.currentSessionSubject.next(null);
    }));
  }

  segmentIdentify(session): void {
    const token = session.token;
    const claims = JSON.parse(atob(token.split('.')[1]));
    // null user ID to ensure we use Anonymous IDs
    const _ = this.segment.identify(null, {
      username: claims.usn,
      email: claims.ema,
      cookie: claims.cki,
    });
  }
}
