import {Component, Injectable, OnInit, OnDestroy} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {ApiUser, Config, ConsoleService, UserRole} from '../console.service';
import {forkJoin, Observable, Subject} from 'rxjs';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';
import {AuthenticationService} from '../authentication.service';
import {DeleteConfirmService} from '../shared/delete-confirm.service';
import {switchMap, takeUntil, tap} from 'rxjs/operators';
import {AuthLoginApiKey, CollectionItem, CollectionList, LayergPortalService} from '../layergPortal.service';
import {NgbModal, NgbModalOptions} from '@ng-bootstrap/ng-bootstrap';
import {ModalLinkContractComponent} from '../components/collection/modalLinkContract/modal-link-contract.component';

@Component({
  templateUrl: './collections.component.html',
  styleUrls: ['./collections.component.scss']
})
export class CollectionsComponent implements OnInit, OnDestroy {
  public readonly systemUserId = '00000000-0000-0000-0000-000000000000';
  public error = '';
  public collectionsCount = 0;
  public collections: Array<CollectionItem> = [];
  public nextCursor = '';
  public prevCursor = '';
  public searchForm: UntypedFormGroup;
  public querySubject: Subject<void>;
  public ongoingQuery = false;

  public limit = 10;
  public page = 1;

  constructor(
      private readonly route: ActivatedRoute,
      private readonly router: Router,
      private readonly consoleService: ConsoleService,
      private readonly layergPortalService: LayergPortalService,
      private readonly authService: AuthenticationService,
      private readonly formBuilder: UntypedFormBuilder,
      private readonly deleteConfirmService: DeleteConfirmService,
      private modalService: NgbModal,
  ) {}

  ngOnInit(): void {
    this.querySubject = new Subject<void>();
    this.searchForm = this.formBuilder.group({
        page: this.page,
        limit: this.limit,
        search: [''],
      filter: [''],
      filter_type: [0], // 0 for all, 1 for tombstones
    });

    const qp = this.route.snapshot.queryParamMap;

    this.f.page.setValue(qp.get('page'));
    this.f.limit.setValue(qp.get('limit'));
    this.f.search.setValue(qp.get('search'));

    this.route.data.subscribe(
        d => {
          // this.search();
          // const configData: Config = d[0][0];
          // const config = JSON.parse(configData?.config);
          // const apiKey = config?.layerg_core?.api_key;
          // const apiKeyID = config?.layerg_core?.api_key_id;
          // const params = {
          //   apiKey,
          //   apiKeyID,
          // };
          this.search();
        },
        err => {
          this.error = err;
        });
  }
  // handleLogin(params: AuthLoginApiKey): any{
  //   try {
  //     this.layergPortalService.login(params).subscribe((d) => {
  //       localStorage.setItem('accessToken', JSON.stringify(d));
  //     }, err => {
  //       console.log(err);
  //       this.error = err?.message;
  //       localStorage.removeItem('accessToken');
  //     });
  //   } catch (e) {
  //     console.log(e);
  //   }
  // }

  ngOnDestroy(): void {
    this.querySubject.next();
    this.querySubject.complete();
  }

  search(): void {
    if (this.ongoingQuery) {
      this.querySubject.next();
    }
    this.ongoingQuery = true;
    const filter = {
        limit: this.f.limit.value || this.limit,
        page: this.f.page.value || this.page,
        search: this.f.search.value || '',
      };

    this.layergPortalService.listCollections(filter)
        .pipe(takeUntil(this.querySubject))
        .subscribe(d => {
          console.log(this.f.search.value);
          this.error = '';

          this.collections.length = 0;
          this.collections.push(...d.data);
          this.collectionsCount = d?.paging?.total;

          this.router.navigate([], {
            relativeTo: this.route,
            queryParams: {
                page: this.f.page.value || this.page,
                limit: this.f.limit.value || this.limit,
                search: this.f.search.value || '',
            },
            queryParamsHandling: 'merge',
          });
          this.ongoingQuery = false;
        }, err => {
          this.error = err;
          this.ongoingQuery = false;
        });
  }

    onPageChange(page: number): void {
        this.f.page.setValue(page);
        this.search();
    }

  cancelQuery(): void {
    this.querySubject.next();
    this.ongoingQuery = false;
  }

  deleteAccount(event, i: number, o: CollectionItem): void {
    this.deleteConfirmService.openDeleteConfirmModal(
        () => {
          event.target.disabled = true;
          event.preventDefault();
          this.error = '';
          this.consoleService.deleteAccount('', o.id, false).subscribe(() => {
            this.error = '';
            this.collections.splice(i, 1);
            this.collectionsCount--;
          }, err => {
            this.error = err;
          });
        }
    );
  }

  deleteAllowed(): boolean {
    // only admin and developers are allowed.
    return this.authService.sessionRole <= UserRole.USER_ROLE_DEVELOPER;
  }

  viewAccount(u: CollectionItem): void {
    this.router.navigate(['/collections', u.id], {relativeTo: this.route});
  }

  get f(): any {
    return this.searchForm.controls;
  }

  // tslint:disable-next-line:typedef
  openLinkContractModal(collection: CollectionItem) {
    const modalOptions: NgbModalOptions = {
      backdrop: true,
      centered: true,
    };
    const modalRef = this.modalService.open(ModalLinkContractComponent, modalOptions);
    modalRef.componentInstance.collectionData = collection; // Truyền dữ liệu vào modal
    modalRef.componentInstance.linkContractUpdated.subscribe(() => {
      this.search();
    });
  }
}

@Injectable({providedIn: 'root'})
export class CollectionsResolver implements Resolve<any> {
  constructor(private readonly consoleService: ConsoleService, private readonly layergPortalService: LayergPortalService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> {
      const filter = {
        limit: route.queryParamMap.get('limit') || 10,
        page: route.queryParamMap.get('page') || 1,
          search: route.queryParamMap.get('search') || '',
      };
      const config = this.consoleService.getConfig('');

      // const collectionList = this.layergPortalService.listCollections(filter);
      return forkJoin([config]);
  }

  // resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> {
  //   return this.consoleService.getConfig('').pipe(
  //     tap((configRes: any) => {
  //       const parsed = JSON.parse(configRes?.config || '{}');
  //       const apiKey = parsed?.layerg_core?.api_key;
  //       const apiKeyID = parsed?.layerg_core?.api_key_id;
  //       const params = {
  //         apiKey,
  //         apiKeyID,
  //       };
  //       this.handleLogin(params);
  //     }),
  //   );
  // }
  // private handleLogin(params: AuthLoginApiKey): void {
  //   try {
  //     this.layergPortalService.login(params).subscribe(
  //       (d) => {
  //         console.log(d);
  //         localStorage.setItem('accessToken', JSON.stringify(d));
  //       },
  //       (err) => {
  //         console.error(err);
  //         localStorage.removeItem('accessToken');
  //       }
  //     );
  //   } catch (e) {
  //     console.error(e);
  //   }
  // }
}
