import {Injectable, Optional} from '@angular/core';
import {HttpClient, HttpHeaders, HttpParams} from '@angular/common/http';
import {CustomHttpParamEncoder} from './console.service';
import {Observable} from 'rxjs';

const DEFAULT_HOST = 'https://w0f43x62-3000.asse.devtunnels.ms/api';
const DEFAULT_TIMEOUT_MS = 5000;

export class LayergEventConfig {
  host!: string;
  timeoutMs!: number;
}
@Injectable({providedIn: 'root'})
export class LayergEventService {
  public layergEvent;

  constructor(private httpClient: HttpClient, @Optional() config: LayergEventConfig) {
    const defaultConfig: LayergEventConfig = {
      host: DEFAULT_HOST,
      timeoutMs: DEFAULT_TIMEOUT_MS,
    };
    this.layergEvent = config || defaultConfig;
  }
  /** List (and optionally filter) accounts. */
  // tslint:disable-next-line:variable-name
  subgraphRegistration(data: any): Observable<any> {
    const urlPath = `/contracts`;
    return this.httpClient.post<any>(this.layergEvent.host + '/api/v1' + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  private getTokenAuthHeaders(): HttpHeaders {
    const accessToken = JSON.parse(localStorage.getItem('accessToken') as string);
    return new HttpHeaders().set('Authorization', 'Bearer ' + accessToken?.accessToken || '');
  }
}
