import {Injectable, Optional} from '@angular/core';
import {HttpClient, HttpHeaders, HttpParams} from '@angular/common/http';
import {CustomHttpParamEncoder} from './console.service';
import {Observable} from 'rxjs';
import {IListNode, INode, INodeForChain} from "./@types/layergEvent";

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
  private getTokenAuthHeaders(): HttpHeaders {
    const accessToken = JSON.parse(localStorage.getItem('accessToken') as string);
    return new HttpHeaders().set('Authorization', 'Bearer ' + accessToken?.accessToken || '');
  }
  subgraphRegistration(data: any): Observable<any> {
    const urlPath = `/contracts`;
    return this.httpClient.post<any>(this.layergEvent.host + '/api/v1' + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  getListNodes(filter?: object): Observable<IListNode> {
    const urlPath = `/gateway/get_list_node`;
    const params = {...filter};
    return this.httpClient.get<IListNode>(this.layergEvent.host + '/v1' + urlPath, { params, headers: this.getTokenAuthHeaders() });
  }
  getNodeForChainId(chainId: string): Observable<INodeForChain> {
    const urlPath = `/gateway/get_node_for_chain`;
    const params = new HttpParams({encoder: new CustomHttpParamEncoder()}).set('chain_id', chainId);
    return this.httpClient.get<INodeForChain>(this.layergEvent.host + '/v1' + urlPath, { params, headers: this.getTokenAuthHeaders() });
  }
}
