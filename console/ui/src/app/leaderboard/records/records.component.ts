import {Component, Injectable, OnInit} from '@angular/core';
import {
  ApiLeaderboardRecord, ApiLeaderboardRecordList,
  ConsoleService, Leaderboard,
  UserRole,
} from '../../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, RouterStateSnapshot} from '@angular/router';
import {AuthenticationService} from '../../authentication.service';
import {Observable} from 'rxjs';
import {DeleteConfirmService} from '../../shared/delete-confirm.service';

@Component({
  templateUrl: './records.component.html',
  styleUrls: ['./records.component.scss']
})
export class LeaderboardRecordsComponent implements OnInit {
  public error = '';

  public leaderboard: Leaderboard;
  public records: Array<ApiLeaderboardRecord> = [];
  public recordsMetadataOpen: Array<boolean> = [];
  public nextCursor = '';
  public prevCursor = '';

  constructor(
    private readonly route: ActivatedRoute,
    private readonly consoleService: ConsoleService,
    private readonly authService: AuthenticationService,
    private readonly deleteConfirmService: DeleteConfirmService,
  ) {}

  ngOnInit(): void {
    this.route.data.subscribe(
      d => {
        this.records.length = 0;
        this.records.push(...d[0].records);
        this.nextCursor = d[0].next_cursor;
        this.prevCursor = d[0].prev_cursor;
        this.recordsMetadataOpen.length = this.records.length;
      },
      err => {
        this.error = err;
      });

    this.route.parent.data.subscribe(
      d => {
        this.leaderboard = d[0];
      },
      err => {
        this.error = err;
      });
  }

  loadRecords(state: number): void {
    let cursor = '';
    switch (state) {
      case -1:
        cursor = this.prevCursor;
        break;
      case 0:
        cursor = '';
        break;
      case 1:
        cursor = this.nextCursor;
        break;
    }

    this.consoleService.listLeaderboardRecords('', this.leaderboard.id, null, 100, cursor, null).subscribe(d => {
      this.error = '';

      this.nextCursor = d.next_cursor;
      this.prevCursor = d.prev_cursor;

      this.records.length = 0;
      this.records.push(...d.records);
      this.recordsMetadataOpen.length = 0; // wipe old records
      this.recordsMetadataOpen.length = this.records.length;
    }, err => {
      this.error = err;
    });
  }

  deleteRecord(event, i: number, r: ApiLeaderboardRecord): void {
    this.deleteConfirmService.openDeleteConfirmModal(
      () => {
        event.target.disabled = true;
        event.preventDefault();
        this.error = '';
        this.consoleService.deleteLeaderboardRecord('', r.leaderboard_id, r.owner_id).subscribe(() => {
          this.error = '';
          this.records.splice(i, 1);
          this.recordsMetadataOpen.splice(i, 1);
        }, err => {
          this.error = err;
        });
      }
    );
  }

  deleteAllowed(): boolean {
    // only admin and developers are allowed.
    return this.authService.sessionRole <= UserRole.USER_ROLE_MAINTAINER;
  }
}

@Injectable({providedIn: 'root'})
export class LeaderboardRecordsResolver implements Resolve<ApiLeaderboardRecordList> {
  constructor(private readonly consoleService: ConsoleService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<ApiLeaderboardRecordList> {
    const leaderboardId = route.parent.paramMap.get('id');
    return this.consoleService.listLeaderboardRecords('', leaderboardId, null, 100, null, null);
  }
}
