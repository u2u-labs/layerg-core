import {Component, Injectable, OnInit} from '@angular/core';
import {
  ApiAccount, ApiFriend, ApiFriendList, ApiUser,
  ConsoleService,
  UserRole,
} from '../../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {AuthenticationService} from '../../authentication.service';
import {Observable} from 'rxjs';
import {DeleteConfirmService} from '../../shared/delete-confirm.service';

@Component({
  templateUrl: './friends.component.html',
  styleUrls: ['./friends.component.scss']
})
export class FriendsComponent implements OnInit {
  public error = '';
  public account: ApiAccount;
  public friends: Array<ApiFriend> = [];

  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly consoleService: ConsoleService,
    private readonly authService: AuthenticationService,
    private readonly deleteConfirmService: DeleteConfirmService,
  ) {}

  ngOnInit(): void {
    this.route.data.subscribe(
      d => {
        this.friends.length = 0;
        this.friends.push(...d[0].friends);
      },
      err => {
        this.error = err;
      });

    this.route.parent.data.subscribe(
      d => {
        this.account = d[0].account;
      },
      err => {
        this.error = err;
      });
  }

  deleteAllowed(): boolean {
    return this.authService.sessionRole <= UserRole.USER_ROLE_MAINTAINER;
  }

  deleteFriend(event, i: number, f: ApiFriend): void {
    this.deleteConfirmService.openDeleteConfirmModal(
      () => {
        event.target.disabled = true;
        event.preventDefault();
        this.error = '';
        this.consoleService.deleteFriend('', this.account.user.id, f.user.id).subscribe(() => {
          this.error = '';
          this.friends.splice(i, 1);
        }, err => {
          this.error = err;
        });
      }
    );
  }

  viewAccount(u: ApiUser): void {
    this.router.navigate(['/accounts', u.id], {relativeTo: this.route});
  }
}

@Injectable({providedIn: 'root'})
export class FriendsResolver implements Resolve<ApiFriendList> {
  constructor(private readonly consoleService: ConsoleService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<ApiFriendList> {
    const userId = route.parent.paramMap.get('id');
    return this.consoleService.getFriends('', userId);
  }
}
