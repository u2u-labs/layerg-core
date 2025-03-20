import {Component, EventEmitter, Input, OnDestroy, OnInit, Output} from '@angular/core';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';
import {
  CollectionItem,
  ICreateCollectible,
  ICreateCollection,
  LayergPortalService
} from '../../../layergPortal.service';
import {UntypedFormBuilder, UntypedFormGroup, Validators, UntypedFormArray} from '@angular/forms';

@Component({
  selector: 'app-collection-modal',
  templateUrl: './modal-create-nft.component.html',
  styleUrls: ['./modal-create-nft.component.scss']
})
export class ModalCreateNftComponent implements OnInit, OnDestroy {
  @Input() collectionData: CollectionItem; // Dữ liệu từ component cha truyền vào
  @Output() collectibleCreated = new EventEmitter<void>();
  public error = '';
  public collectibleForm: UntypedFormGroup;
  selectedFile: File | null = null;
  imagePreview: string | ArrayBuffer | null = null;
  constructor(
    public activeModal: NgbActiveModal,
    private readonly layergPortalService: LayergPortalService,
    private readonly formBuilder: UntypedFormBuilder,
    ) {}

  ngOnInit(): void {
    this.collectibleForm = this.formBuilder.group({
      name: ['', Validators.required],
      description: ['', [Validators.maxLength(500)]],
      tokenId: [''],
      avatarUrl: ['', Validators.required],
      quantity: ['', Validators.required],
      attributes: this.formBuilder.array([
        this.formBuilder.group({
          trait_type: [''],
          value: [''],
        }),
      ]),
    });
  }
  get f(): any {
    return this.collectibleForm.controls;
  }
  get attributes(): UntypedFormArray {
    return this.collectibleForm.get('attributes') as UntypedFormArray;
  }
  addAttribute(): void {
    this.attributes.push(this.formBuilder.group({
      trait_type: ['', Validators.required],
      value: ['', Validators.required],
    }));
  }
  removeAttribute(index: number): void {
    this.attributes.removeAt(index);
  }
  ngOnDestroy(): void {}
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

    this.layergPortalService.uploadNft(formData).subscribe(
      (data) => {
        console.log('Image uploaded', data);
        // callback(data.url);
        this.f.avatarUrl.setValue(data[0]);
        console.log(this.f.avatarUrl.value);
      },
      (error) => {
        console.error('Error uploading image', error);
        alert('Error uploading image');
      }
    );
  }
  createCollectible(): void {
    try {
      this.error = '';
      if (this.collectibleForm.invalid) {
        return;
      }
      const body: ICreateCollectible = {
        name: this.f.name.value,
        description: this.f.description.value,
        avatarUrl: this.f.avatarUrl.value,
        // tokenId: this.f.tokenId.value.toString() || '',
        collectionId: this.collectionData.id,
        quantity: this.f.quantity.value.toString(),
        metadata: {
          metadata: {
            attributes: this.attributes.value,
          }
        },
        media: {
          S3Url: this.f.avatarUrl.value
        }
      };
      if (this.f.tokenId.value.toString()) {
        body.tokenId = this.f.tokenId.value.toString();
      }
      this.layergPortalService.createCollectibleNft(body).subscribe((d) => {
        this.activeModal.dismiss();
        this.collectibleCreated.emit();
      }, err => {
        console.log(err);
        this.error = err;
      });
    } catch (e) {
      console.log(e);
      this.error = e;
    } finally {
      // this.activeModal.dismiss();
    }
  }
}
