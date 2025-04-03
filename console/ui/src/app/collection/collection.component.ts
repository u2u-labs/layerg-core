import {Component, Injectable, OnInit, OnDestroy} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {ApiUser, Config, ConsoleService, UserRole} from '../console.service';
import {forkJoin, Observable, Subject} from 'rxjs';
import {UntypedFormBuilder, UntypedFormGroup} from '@angular/forms';
import {AuthenticationService} from '../authentication.service';
import {DeleteConfirmService} from '../shared/delete-confirm.service';
import {takeUntil} from 'rxjs/operators';
import {CollectionItem, CollectionList, LayergPortalService} from '../layergPortal.service';

@Component({
  templateUrl: './collection.component.html',
  styleUrls: ['./collection.component.scss']
})
export class CollectionComponent implements OnInit {
  public readonly systemUserId = '00000000-0000-0000-0000-000000000000';
  public error = '';
  public collectionsCount = 0;
  public accounts: Array<CollectionItem> = [];
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
  ) {}

  ngOnInit(): void {

    this.route.data.subscribe(
      d => {
      },
      err => {
        this.error = err;
      });
  }

  viewAccount(u: CollectionItem): void {
    this.router.navigate(['/collections', u.id], {relativeTo: this.route});
  }
}

@Injectable({providedIn: 'root'})
export class CollectionResolver implements Resolve<CollectionList> {
  constructor(private readonly consoleService: ConsoleService, private readonly layergPortalService: LayergPortalService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> {
    const userId = route.paramMap.get('id');
    const config = this.consoleService.getConfig('');
    const getDetailCollection = this.layergPortalService.detailCollection(userId);
    return forkJoin([config, getDetailCollection]);
  }
}
