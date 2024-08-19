import {Component, Injectable, OnInit} from '@angular/core';
import {
  ConsoleAddGroupUsersBody as AddGroupUsersRequest,
  ApiGroup,
  ApiGroupUserList,
  ConsoleService,
  GroupUserListGroupUser, 
  ConsoleUpdateAccountBody as UpdateAccountRequest,
  UserGroupListUserGroup,
  UserRole
} from '../../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {AuthenticationService} from '../../authentication.service';
import {Observable} from 'rxjs';
import {UntypedFormBuilder, UntypedFormGroup} from "@angular/forms";
import {resolve} from "@angular/compiler-cli/src/ngtsc/file_system";

@Component({
  templateUrl: './groupMembers.component.html',
  styleUrls: ['./groupMembers.component.scss']
})
export class GroupMembersComponent implements OnInit {
  public error = '';
  public group: ApiGroup;
  public members: Array<GroupUserListGroupUser> = [];
  public activeState = 'Add Member';
  public readonly states = ['Add Member', 'Join'];
  public addForm: UntypedFormGroup;

  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly consoleService: ConsoleService,
    private readonly formBuilder: UntypedFormBuilder,
    private readonly authService: AuthenticationService,
  ) {
    this.router.routeReuseStrategy.shouldReuseRoute = () => false;
    this.router.onSameUrlNavigation = 'reload'
    this.addForm = this.formBuilder.group({
      ids: [''],
    });
  }

  ngOnInit(): void {
    this.route.data.subscribe(
      d => {
        this.members.length = 0;
        this.members.push(...d[0].group_users);
      },
      err => {
        this.error = err;
      });
    this.route.parent.data.subscribe(
      d => {
        this.group = d[0];
      },
      err => {
        this.error = err;
      });
  }

  editionAllowed() {
    return this.authService.sessionRole <= UserRole.USER_ROLE_MAINTAINER;
  }

  deleteGroupUser(event, i: number, f: GroupUserListGroupUser) {
    event.target.disabled = true;
    event.preventDefault();
    this.error = '';
    this.consoleService.deleteGroupUser('', f.user.id, this.group.id).subscribe(() => {
      this.members.splice(i, 1)
    }, err => {
      this.error = err;
    })
  }

  demoteGroupUser(event, i: number, f: GroupUserListGroupUser) {
    this.error = '';
    this.consoleService.demoteGroupMember('', this.group.id, f.user.id).subscribe(() => {
      this.members[i].state++;
    }, err => {
      this.error = err;
    })
  }

  promoteGroupUser(event, i: number, f: GroupUserListGroupUser) {
    this.error = '';
    this.consoleService.promoteGroupMember('', this.group.id, f.user.id).subscribe(() => {
      this.members[i].state--;
    }, err => {
      this.error = err;
    })
  }

  viewAccount(g: GroupUserListGroupUser): void {
    this.router.navigate(['/accounts', g.user.id], {relativeTo: this.route});
  }

  add(): void {
    let body: AddGroupUsersRequest = {ids: this.f.ids.value, join_request: this.activeState === 'Join'};
    this.consoleService.addGroupUsers('', this.group.id, body).subscribe(() => {
      this.error = '';
      // refresh
      this.router.navigate([this.router.url])
    }, err => {
      this.error = err;
    });
  }

  get f(): any {
    return this.addForm.controls;
  }
}

@Injectable({providedIn: 'root'})
export class GroupMembersResolver implements Resolve<ApiGroupUserList> {
  constructor(private readonly consoleService: ConsoleService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<ApiGroupUserList> {
    const groupId = route.parent.paramMap.get('id');
    return this.consoleService.getMembers('', groupId);
  }
}
