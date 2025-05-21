import {Component, Injectable, OnDestroy, OnInit} from '@angular/core';
import {Observable, Subject} from 'rxjs';
import {Config, ConsoleService} from '../../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {ICreateCollection, LayergPortalService} from '../../layergPortal.service';
import {AuthenticationService} from '../../authentication.service';
import {UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';

@Component({
  templateUrl: './createCollection.component.html',
  styleUrls: ['./createCollection.component.scss']
})
export class CreateCollectionComponent implements OnInit, OnDestroy {
  public error = '';
  public collectionForm: UntypedFormGroup;
  selectedFile: File | null = null;
  imagePreview: string | ArrayBuffer | null = null;
  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly layergPortalService: LayergPortalService,
    private readonly authService: AuthenticationService,
    private readonly formBuilder: UntypedFormBuilder,
  ) {}
  ngOnInit(): void {
    this.collectionForm = this.formBuilder.group({
      name: ['', [Validators.required, Validators.minLength(5)]],
      description: ['', [Validators.maxLength(255)]],
      avatarUrl: ['', Validators.required],
    });
    this.route.data.subscribe(
      d => {
        // const data = JSON.parse(d[0].config);
        // console.log(data);
      },
      err => {
        this.error = err;
      });
  }
  get f(): any {
    return this.collectionForm.controls;
  }
  ngOnDestroy(): void {
    // this.querySubject.next();
    // this.querySubject.complete();
  }
  onFileSelected(event: Event): void {
    const fileInput = event.target as HTMLInputElement;
    if (fileInput.files && fileInput.files.length > 0) {
      this.selectedFile = fileInput.files[0];

      // Preview the image
      const reader = new FileReader();
      reader.onload = () => {
        this.imagePreview = reader.result;
      };
      reader.readAsDataURL(this.selectedFile);
      this.uploadImage();
    }
  }
  uploadImage(): void {
    if (!this.selectedFile) {
      alert('Please select an image first!');
      return;
    }

    const formData = new FormData();
    formData.append('files', this.selectedFile);

    this.layergPortalService.uploadImage(formData).subscribe(
      (data) => {
        console.log('Image uploaded', data);
        // callback(data.url);
        this.f.avatarUrl.setValue(this.layergPortalService.getIpfsUrl(data.fileHashes[0]));
        console.log(this.f.avatarUrl.value);
      },
      (error) => {
        console.error('Error uploading image', error);
        alert('Error uploading image');
      }
    );
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
        projectId: JSON.parse(localStorage.getItem('accessToken') as string)?.projectId || null,
      };
      this.layergPortalService.createCollections(body).subscribe((d) => {
        this.router.navigate(['/collections']);
      }, err => {
        console.log(err);
        this.error = err;
      });
    } catch (e) {
      console.log(e);
      this.error = e;
    }
  }
  handleBack(): void {
    this.router.navigate(['/collections']);
  }
}

@Injectable({providedIn: 'root'})

export class CreateCollectionResolver implements Resolve<Config> {
  constructor(private readonly consoleService: ConsoleService, private readonly layergPortalService: LayergPortalService) {}


  resolve(route: ActivatedRouteSnapshot, state: RouterStateSnapshot): Observable<Config> {
    return this.consoleService.getConfig('');
  }
}
