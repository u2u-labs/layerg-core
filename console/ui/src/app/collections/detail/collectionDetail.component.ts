import {AfterViewInit, Component, ElementRef, Injectable, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {Observable, Subject, forkJoin} from 'rxjs';
import {Config, ConsoleService} from '../../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {ICreateCollection, LayergPortalService} from '../../layergPortal.service';
import {AuthenticationService} from '../../authentication.service';
import {ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';
import {JSONEditor, Mode, toTextContent} from 'vanilla-jsoneditor';

@Component({
  selector: 'app-collection-detail',
  templateUrl: './collectionDetail.component.html',
  styleUrls: ['./collectionDetail.component.scss'],
})
export class CollectionDetailComponent implements OnInit, AfterViewInit {
  @ViewChild('editor') private editor: ElementRef<HTMLElement>;

  public error = '';
  private jsonEditor: JSONEditor;
  public collectionForm: UntypedFormGroup;
  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly layergPortalService: LayergPortalService,
    private readonly authService: AuthenticationService,
    private readonly formBuilder: UntypedFormBuilder,
  ) {}
  ngOnInit(): void {
    this.collectionForm = this.formBuilder.group({
      name: ['', Validators.required],
      description: ['', [Validators.maxLength(255)]],
      avatarUrl: ['', Validators.required],
      projectId: ['', Validators.required],
    });
    this.route.data.subscribe(
      d => {
        console.log(d);
      },
      err => {
        this.error = err;
      });
  }
  get f(): any {
    return this.collectionForm.controls;
  }
  ngAfterViewInit(): void {
  }
  createCollection(): void {
    try {
      this.error = '';
      if (this.collectionForm.invalid) {
        return;
      }
      const body: ICreateCollection = {
        name: this.f.name.value,
        description: this.f.description.value,
        avatarUrl: this.f.avatarUrl.value,
        projectId: this.f.projectId.value,
      };
      console.log(body);
    } catch (e) {
      this.error = e;
    }
  }
  handleBack(): void {
    this.router.navigate(['/collections']);
  }
}

@Injectable({providedIn: 'root'})

export class CollectionDetailResolver implements Resolve<any> {
  constructor(private readonly consoleService: ConsoleService, private readonly layergPortalService: LayergPortalService) {}

  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<any> {
    const userId = route.paramMap.get('id');
    const config = this.consoleService.getConfig('');
    const getDetailCollection = this.layergPortalService.detailCollection(userId);
    return forkJoin([config, getDetailCollection]);
  }
}
