import {Injectable, Optional} from '@angular/core';
import {HttpClient, HttpHeaders, HttpParams} from '@angular/common/http';
import {CustomHttpParamEncoder} from './console.service';
import {Observable} from 'rxjs';

export interface Project {
  id: string;
  createdAt: string;
  updatedAt: string;
  isEnabled: boolean;
  countFav: number;
  platform: string[];
  name: string;
  gameIcon: string;
  banner: string;
  apiKeyID: string;
  telegram: string;
  facebook: string;
  instagram: string;
  discord: string;
  twitter: string;
  nameSlug: string;
  avatar: string;
  description: string;
  information: string;
  policy: string;
  version: string;
  slideShow: string[];
  totalReview: number;
  totalRating: number;
  slug: any;
  isRcm: boolean;
  userId: any;
}

export interface SmartContract {
  id: string;
  updatedAt: string;
  contractAddress: string;
  contractType: string;
  networkID: number;
  contractName: string;
  tokenSymbol: string;
  totalSupply: any;
  collectionId: string;
  deployedAt: string;
  nameSlug: string;
}

/** A user in the server. */
export interface CollectionItem {
  // The Apple Sign In ID in the user's account.
  id: string;
  createdAt: string;
  updatedAt: string;
  name: string;
  nameSlug: string;
  description: string;
  avatarUrl: string;
  projectId: string;
  project: Project;
  totalAssets: number;
  slug: string;
  SmartContract: SmartContract[];
}
/** A list of users. */
export interface CollectionList {
  paging: Paging;
  // A list of users.
  data?: Array<CollectionItem>;
}

export interface ICreateCollection {
  name: string;
  description: string;
  avatarUrl: string;
  projectId: string;
}
export interface IUpdateCollection {
  name: string;
  description: string;
  avatarUrl: string;
  projectId: string;
}

export interface Paging {
  total: number;
  limit: number;
  page: number;
}
export interface AuthLoginApiKey {
  apiKey: string;
  apiKeyID: string;
}

export interface ICreateCollectible {
  name: string;
  description: string;
  avatarUrl: string;
  tokenId?: string;
  collectionId: string;
  quantity: string;
  metadata?: {
    metaId?: string;
    metadata?: {
      attributes: { 'trait_type': string, 'value': string };
    }
  };
  media?: {
    mediaId?: string;
    S3Url?: string;
  };
}

// Collectible Item NFT
export interface ICollectibleItem {
  id: string;
  tokenId: string;
  collectionId: string;
  name: string;
  description: string;
  createdAt: string;
  updatedAt: string;
  mediaStorageId: string;
  metaDataId: string;
  nameSlug: string;
  slug: string;
  quantity: number;
  collection: {
    id: string
    createdAt: string
    updatedAt: string
    totalAssets: number
    name: string
    description: string
    avatarUrl: string
    projectId: string
    nameSlug: string
    slug: string
  };
  media: {
    id: string
    S3Url: string
    IPFSUrl: any
    apiKeyID: string
  };
  metadata: {
    id: string
    metadata: {
      attributes: [{
        value: string
        trait_type: string
      }]
    }
    IPFSUrl: any
    apiKeyID: string
  };
}
export interface CollectibleList {
  paging: Paging;
  // A list of users.
  data?: Array<ICollectibleItem>;
}

const DEFAULT_HOST = 'https://w0f43x62-3000.asse.devtunnels.ms/api';
const DEFAULT_TIMEOUT_MS = 5000;

export class LayergPortalConfig {
  host!: string;
  timeoutMs!: number;
}
@Injectable({providedIn: 'root'})
export class LayergPortalService {
  private readonly layerg;

  constructor(private httpClient: HttpClient, @Optional() config: LayergPortalConfig) {
    const defaultConfig: LayergPortalConfig = {
      host: DEFAULT_HOST,
      timeoutMs: DEFAULT_TIMEOUT_MS,
    };
    this.layerg = config || defaultConfig;
  }
  /** List (and optionally filter) accounts. */
  // tslint:disable-next-line:variable-name
  login(data: AuthLoginApiKey): Observable<any> {
    const urlPath = `/auth/login`;
    return this.httpClient.post<any>(this.layerg.host + urlPath, data);
  }
  listCollections(filter?: object): Observable<CollectionList> {
    const urlPath = `/collection`;
    const params = {...filter};
    return this.httpClient.get<CollectionList>(this.layerg.host + urlPath, { params, headers: this.getTokenAuthHeaders() });
  }
  createCollections(data: ICreateCollection): Observable<ICreateCollection> {
    const urlPath = `/collection`;
    return this.httpClient.post<ICreateCollection>(this.layerg.host + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  updateCollections(id: string, data: IUpdateCollection): Observable<IUpdateCollection> {
    id = encodeURIComponent(String(id));
    const urlPath = `/collection/${id}`;
    return this.httpClient.put<IUpdateCollection>(this.layerg.host + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  detailCollection(id: string): Observable<CollectionItem> {
    id = encodeURIComponent(String(id));
    const urlPath = `/collection/${id}`;
    return this.httpClient.get<CollectionItem>(this.layerg.host + urlPath, { headers: this.getTokenAuthHeaders() });
  }
  uploadImage(data: FormData): Observable<any> {
    const urlPath = `/common/upload-ipfs`;
    return this.httpClient.post<{fileHashes: string | string[]}>(this.layerg.host + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  getIpfsUrl(hash: string): string {
    return `${this.layerg.host}/common/ipfs-serve?ipfsPath=${hash}`;
  }
  uploadNft(data: FormData): Observable<any> {
    const urlPath = `/common/upload`;
    return this.httpClient.post<{fileHashes: string | string[]}>(this.layerg.host + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  createCollectibleNft(data: ICreateCollectible): Observable<ICreateCollectible> {
    const urlPath = `/assets/create`;
    return this.httpClient.post<ICreateCollectible>(this.layerg.host + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  updateCollectibleNft(id: string, data: ICreateCollectible): Observable<ICreateCollectible> {
    const urlPath = `/assets/${id}`;
    return this.httpClient.put<ICreateCollectible>(this.layerg.host + urlPath, data, { headers: this.getTokenAuthHeaders() });
  }
  listCollectibles(filter?: object): Observable<CollectibleList> {
    const urlPath = `/assets`;
    const params = {...filter};
    return this.httpClient.get<CollectibleList>(this.layerg.host + urlPath, { params, headers: this.getTokenAuthHeaders() });
  }
  detailCollectible(id: string): Observable<ICollectibleItem> {
    const urlPath = `/assets/${id}`;
    return this.httpClient.get<ICollectibleItem>(this.layerg.host + urlPath, { headers: this.getTokenAuthHeaders() });
  }
  private getTokenAuthHeaders(): HttpHeaders {
    const accessToken = JSON.parse(localStorage.getItem('accessToken') as string);
    return new HttpHeaders().set('Authorization', 'Bearer ' + accessToken?.accessToken || '');
  }
}
