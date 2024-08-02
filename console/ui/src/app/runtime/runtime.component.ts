import {Component, Injectable, OnDestroy, OnInit} from '@angular/core';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, RouterStateSnapshot} from '@angular/router';
import {ConsoleService, RuntimeInfo} from '../console.service';
import {Observable} from 'rxjs';

@Component({
  templateUrl: './runtime.component.html',
  styleUrls: ['./runtime.component.scss']
})
export class RuntimeComponent implements OnInit, OnDestroy {
  public error = '';
  public runtimeInfo: RuntimeInfo;

  constructor(
    private readonly route: ActivatedRoute,
    private readonly consoleService: ConsoleService,
  ) {}

  ngOnInit(): void {
    this.route.data.subscribe(
      d => {
        this.runtimeInfo = d[0];
      },
      err => {
        this.error = err;
      });
  }

  ngOnDestroy(): void {
  }
}

@Injectable({providedIn: 'root'})
export class RuntimeResolver implements Resolve<RuntimeInfo> {
  constructor(private readonly consoleService: ConsoleService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<RuntimeInfo> {
    return this.consoleService.getRuntime('');
  }
}
