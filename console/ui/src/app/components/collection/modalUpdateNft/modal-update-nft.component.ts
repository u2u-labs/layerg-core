import {Component, EventEmitter, Input, OnDestroy, OnInit, Output} from '@angular/core';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';
import {
  CollectionItem, ICollectibleItem,
  ICreateCollectible,
  ICreateCollection,
  LayergPortalService
} from '../../../layergPortal.service';
import {UntypedFormBuilder, UntypedFormGroup, Validators, UntypedFormArray} from '@angular/forms';

@Component({
  selector: 'app-collection-modal',
  templateUrl: './modal-update-nft.component.html',
  styleUrls: ['./modal-update-nft.component.scss']
})
export class ModalUpdateNftComponent implements OnInit, OnDestroy {
  @Input() collectionData: CollectionItem; // Dữ liệu từ component cha truyền vào
  @Input() collectibleData: ICollectibleItem; // Dữ liệu từ component cha truyền vào
  @Output() collectibleUpdated = new EventEmitter<void>();
  public error = '';
  public collectibleForm: UntypedFormGroup;
  selectedFile: File | null = null;
  imagePreview: string | ArrayBuffer | null = null;
  public item: ICollectibleItem;
  constructor(
    public activeModal: NgbActiveModal,
    private readonly layergPortalService: LayergPortalService,
    private readonly formBuilder: UntypedFormBuilder,
    ) {}

  ngOnInit(): void {
    this.handleDetail();
    this.collectibleForm = this.formBuilder.group({
      name: ['', Validators.required],
      description: ['', [Validators.maxLength(500)]],
      tokenId: ['', Validators.required],
      avatarUrl: ['', Validators.required],
      quantity: ['0', Validators.required],
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
  updateCollectible(): void {
    try {
      this.error = '';
      if (this.collectibleForm.invalid) {
        return;
      }
      const id = `${this.collectionData?.id}/${this.collectibleData?.id}`;
      const body: ICreateCollectible = {
        name: this.f.name.value,
        description: this.f.description.value,
        avatarUrl: this.f.avatarUrl.value,
        tokenId: this.f.tokenId.value.toString(),
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
      this.layergPortalService.updateCollectibleNft(id, body).subscribe((d) => {
        this.activeModal.dismiss();
        this.collectibleUpdated.emit();
      }, err => {
        console.log(err);
        this.error = err;
      });
    } catch (e) {
      console.log(e);
      this.error = e;
    } finally {
    }
  }
  handleDetail(): void {
    try {
      this.error = '';
      const id = `${this.collectionData?.id}/${this.collectibleData?.id}`;
      this.layergPortalService.detailCollectible(id).subscribe((d) => {
        this.f.name.setValue(d.name);
        this.f.description.setValue(d.description);
        this.f.tokenId.setValue(d.tokenId);
        this.f.quantity.setValue(d.quantity);
        this.imagePreview = d.media.S3Url || d.media.IPFSUrl;
        this.f.avatarUrl.setValue(d.media.S3Url || d.media.IPFSUrl);
        // Clear the existing attributes
        this.attributes.clear();

        // Map the attributes from the data
        d.metadata.metadata.attributes.forEach((attr: { trait_type: string; value: string }) => {
          this.attributes.push(this.formBuilder.group({
            trait_type: [attr.trait_type, Validators.required],
            value: [attr.value, Validators.required],
          }));
        });
        this.item = d;
      }, err => {
        console.log(err);
        this.error = err;
      });
    } catch (e) {
      console.log(e);
      this.error = e;
    } finally {
    }
  }
}
