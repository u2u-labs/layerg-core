import {Component, Injectable, OnInit} from '@angular/core';
import {ConsoleService, Leaderboard, UserRole} from '../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {Observable} from 'rxjs';
import {AuthenticationService} from '../authentication.service';
import {DeleteConfirmService} from '../shared/delete-confirm.service';

@Component({
  templateUrl: './leaderboard.component.html',
  styleUrls: ['./leaderboard.component.scss']
})
export class LeaderboardComponent implements OnInit {
  public leaderboard: Leaderboard;
  public error = '';

  public views = [
    {label: 'Details', path: 'details'},
    {label: 'Records', path: 'records'},
  ];

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
        this.leaderboard = d[0];
      },
      err => {
        this.error = err;
      });
  }

  deleteLeaderboard(event): void {
    this.deleteConfirmService.openDeleteConfirmModal(
      () => {
        event.target.disabled = true;
        this.error = '';
        this.consoleService.deleteLeaderboard('', this.leaderboard.id).subscribe(() => {
          this.error = '';
          this.router.navigate(['/leaderboards']);
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
}

@Injectable({providedIn: 'root'})
export class LeaderboardResolver implements Resolve<Leaderboard> {
  constructor(private readonly consoleService: ConsoleService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<Leaderboard> {
    const leaderboardId = route.paramMap.get('id');
    return this.consoleService.getLeaderboard('', leaderboardId);
  }
}
