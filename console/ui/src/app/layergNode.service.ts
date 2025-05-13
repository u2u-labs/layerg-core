import {Injectable, Optional} from '@angular/core';
import {HttpClient, HttpHeaders, HttpParams} from '@angular/common/http';
import {Observable} from 'rxjs';
import {SubgraphRegistrationData} from './@types/layergNode';

const DEFAULT_HOST = '';
const DEFAULT_TIMEOUT_MS = 5000;

export class LayergNodeConfig {
  host!: string;
  timeoutMs!: number;
}
@Injectable({providedIn: 'root'})
export class LayergNodeService {
  public layergNode;

  constructor(private httpClient: HttpClient, @Optional() config: LayergNodeConfig) {
    const defaultConfig: LayergNodeConfig = {
      host: DEFAULT_HOST,
      timeoutMs: DEFAULT_TIMEOUT_MS,
    };
    this.layergNode = config || defaultConfig;
  }
  /** List (and optionally filter) accounts. */
  // tslint:disable-next-line:variable-name
  private getTokenAuthHeaders(): HttpHeaders {
    const accessToken = JSON.parse(localStorage.getItem('accessToken') as string);
    return new HttpHeaders().set('Authorization', 'Bearer ' + accessToken?.accessToken || '');
  }
  subgraphRegistration(data: SubgraphRegistrationData): Observable<any> {
    const urlPath = `/contracts`;
    return this.httpClient.post<any>(this.layergNode.host + '/api/v1' + urlPath, data);
  }
}
