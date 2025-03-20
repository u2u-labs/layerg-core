import {AfterViewInit, Component, ElementRef, Injectable, OnDestroy, OnInit, ViewChild} from '@angular/core';
import {Observable, Subject, forkJoin} from 'rxjs';
import {Config, ConsoleService} from '../../console.service';
import {ActivatedRoute, ActivatedRouteSnapshot, Resolve, Router, RouterStateSnapshot} from '@angular/router';
import {CollectionItem, ICollectibleItem, ICreateCollection, LayergPortalService} from '../../layergPortal.service';
import {AuthenticationService} from '../../authentication.service';
import {ReactiveFormsModule, UntypedFormBuilder, UntypedFormGroup, Validators} from '@angular/forms';
import {JSONEditor, Mode, toTextContent} from 'vanilla-jsoneditor';
import {NgbModal, NgbModalOptions} from '@ng-bootstrap/ng-bootstrap';
import {ModalCreateNftComponent} from '../../components/collection/modalCreateNft/modal-create-nft.component';
import {ModalUpdateNftComponent} from '../../components/collection/modalUpdateNft/modal-update-nft.component';

@Component({
  templateUrl: './collectionDetail.component.html',
  styleUrls: ['./collectionDetail.component.scss'],
})
export class CollectionDetail1Component implements OnInit, AfterViewInit {
  constructor(
    private readonly route: ActivatedRoute,
    private readonly router: Router,
    private readonly layergPortalService: LayergPortalService,
    private readonly authService: AuthenticationService,
    private readonly formBuilder: UntypedFormBuilder,
    private modalService: NgbModal
  ) {}
  get f(): any {
    return this.collectionForm.controls;
  }
  get c(): any {
    return this.searchForm.controls;
  }
  @ViewChild('editor') private editor: ElementRef<HTMLElement>;

  public error = '';
  private jsonEditor: JSONEditor;
  public collectionForm: UntypedFormGroup;
  public collection: CollectionItem;
  public config: Config;
  selectedFile: File | null = null;
  imagePreview: string | ArrayBuffer | null = null;
  public limit = 10;
  public page = 1;
  public querySubject: Subject<void>;
  public ongoingQuery = false;
  public searchForm: UntypedFormGroup;
  public collectibleCount = 0;
  public collectibles: Array<ICollectibleItem> = [];

  protected readonly Number = Number;
  protected readonly String = String;
  ngOnInit(): void {
    this.collectionForm = this.formBuilder.group({
      name: ['', Validators.required],
      description: ['', [Validators.maxLength(255)]],
      avatarUrl: ['', Validators.required],
    });
    this.searchForm = this.formBuilder.group({
      page: this.page,
      limit: this.limit,
      search: [''],
      filter: [''],
      collectionId: [''],
    });
    this.route.parent.data.subscribe(
      d => {
        this.config = d[0][0];
        this.collection = d[0][1];
        this.collectionForm.patchValue(this.collection);
        this.imagePreview = this.collection.avatarUrl;
        this.f.avatarUrl.setValue(this.collection.avatarUrl);
        this.getListCollectible();
      },
      err => {
        this.error = err;
      });

  }
  ngAfterViewInit(): void {
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
      this.layergPortalService.updateCollections(this.collection?.id, body).subscribe((d) => {
        this.router.navigate(['/collections']);
      }, err => {
        console.log(err);
        this.error = err;
      });
    } catch (e) {
      this.error = e;
    }
  }
  handleBack(): void {
    this.router.navigate(['/collections']);
  }
  getListCollectible(): void {
    const filter = {
      limit: this.c.limit.value || this.limit,
      page: this.c.page.value || this.page,
      search: this.c.search.value || '',
      collectionId: this.collection?.id,
    };

    this.layergPortalService.listCollectibles(filter)
      // .pipe(takeUntil(this.querySubject))
      .subscribe(d => {
        this.error = '';

        this.collectibles.length = 0;
        this.collectibles.push(...d.data);
        this.collectibleCount = d?.paging?.total;

        // this.router.navigate([], {
        //   relativeTo: this.route,
        //   queryParams: {
        //     page: this.f.page.value || this.page,
        //     limit: this.f.limit.value || this.limit,
        //     search: this.f.search.value || '',
        //   },
        //   queryParamsHandling: 'merge',
        // });
        // this.ongoingQuery = false;
      }, err => {
        this.error = err;
        this.ongoingQuery = false;
      });
  }
  onPageChange(page: number): void {
    this.c.page.setValue(page);
    this.getListCollectible();
  }
  changeLimit(event: Event): void {
    const limit = +(event.target as HTMLSelectElement).value;
    this.c.limit.setValue(limit);
    this.getListCollectible();
  }
  viewDetailNft(collectible: ICollectibleItem, collection: CollectionItem): void {
    const modalOptions: NgbModalOptions = {
      backdrop: true,
      centered: true,
    };
    const modalRef = this.modalService.open(ModalUpdateNftComponent, modalOptions);
    modalRef.componentInstance.collectionData = collection; // Truyền dữ liệu vào modal
    modalRef.componentInstance.collectibleData = collectible; // Truyền dữ liệu vào modal
    modalRef.componentInstance.collectibleUpdated.subscribe(() => {
      this.getListCollectible();
    });
  }
  // tslint:disable-next-line:typedef
  openModal(collection: CollectionItem) {
    const modalOptions: NgbModalOptions = {
      backdrop: true,
      centered: true,
    };
    const modalRef = this.modalService.open(ModalCreateNftComponent, modalOptions);
    modalRef.componentInstance.collectionData = collection; // Truyền dữ liệu vào modal
    modalRef.componentInstance.collectibleCreated.subscribe(() => {
      this.getListCollectible();
    });
  }
}
