import {Component, Injectable, OnInit} from '@angular/core';
import {UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';
import {ActivatedRoute, ActivatedRouteSnapshot, CanActivate, Router, RouterStateSnapshot} from '@angular/router';
import {AuthenticationService} from '../authentication.service';
import {SegmentService} from 'ngx-segment-analytics';
import {environment} from "../../environments/environment";

@Component({
  templateUrl: './login.component.html',
  styleUrls: ['./login.component.scss']
})
export class LoginComponent implements OnInit {
  public error = '';
  public loginForm!: UntypedFormGroup;
  public submitted!: boolean;
  private returnUrl!: string;

  constructor(
    private segment: SegmentService,
    private readonly formBuilder: UntypedFormBuilder,
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly authenticationService: AuthenticationService
  ) {}

  ngOnInit(): void {
    if (!environment.nt) {
      this.segment.page('/login');
    }
    this.loginForm = this.formBuilder.group({
      username: ['', Validators.compose([Validators.required])],
      password: ['', Validators.compose([Validators.required, Validators.minLength(8)])],
    });
    this.returnUrl = this.route.snapshot.queryParams.next || '/';
  }

  onSubmit(): void {
    this.submitted = true;
    this.error = '';
    if (this.loginForm.invalid) {
      return;
    }
    this.authenticationService.login(this.f.username.value, this.f.password.value)
      .subscribe(session => {
        this.loginForm.reset();
        this.submitted = false;
        this.router.navigate([this.returnUrl]);
      }, err => {this.error = err; this.submitted = false; });
  }

  get f(): any {
    return this.loginForm.controls;
  }
}

@Injectable({providedIn: 'root'})
export class LoginGuard implements CanActivate {
  constructor(private readonly authService: AuthenticationService, private readonly router: Router) {}

  canActivate(next: ActivatedRouteSnapshot, state: RouterStateSnapshot): boolean {
    if (this.authService.currentSessionValue) {
      const _ = this.router.navigate(['/']);
      return false;
    }

    return true;
  }
}
