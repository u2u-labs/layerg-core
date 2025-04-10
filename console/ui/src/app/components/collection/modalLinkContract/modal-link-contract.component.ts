import {Component, EventEmitter, Input, OnDestroy, OnInit, Output} from '@angular/core';
import { NgbActiveModal } from '@ng-bootstrap/ng-bootstrap';
import {
  CollectionItem, ICollectibleItem, ILinkContract, IUpdateCollection,
  LayergPortalService
} from '../../../layergPortal.service';
import {UntypedFormBuilder, UntypedFormGroup, Validators, UntypedFormArray} from '@angular/forms';

@Component({
  selector: 'app-collection-link-contract-modal',
  templateUrl: './modal-link-contract.component.html',
  styleUrls: ['./modal-link-contract.component.scss']
})
export class ModalLinkContractComponent implements OnInit, OnDestroy {
  @Input() collectionData: CollectionItem; // Dữ liệu từ component cha truyền vào
  @Output() linkContractUpdated = new EventEmitter<void>();
  public error = '';
  public linkContractForm: UntypedFormGroup;
  constructor(
    public activeModal: NgbActiveModal,
    private readonly layergPortalService: LayergPortalService,
    private readonly formBuilder: UntypedFormBuilder,
    ) {}

  ngOnInit(): void {
    // this.handleDetail();

    this.linkContractForm = this.formBuilder.group({
      contractAddress: ['', Validators.required],
      contractType: ['', Validators.required],
      networkID: ['', Validators.required],
      tokenSymbol: ['', Validators.required],
    });
    if (this.collectionData?.SmartContract && this.collectionData?.SmartContract.length) {
      const smc: ILinkContract = this.collectionData?.SmartContract[0];
      this.f.contractAddress.setValue(smc.contractAddress);
      this.f.contractType.setValue(smc.contractType);
      this.f.networkID.setValue(smc.networkID);
      this.f.tokenSymbol.setValue(smc.tokenSymbol);
    }
  }
  get f(): any {
    return this.linkContractForm.controls;
  }
  ngOnDestroy(): void {}
  updateLinkContract(): void {
    try {
      this.error = '';
      if (this.linkContractForm.invalid) {
        return;
      }
      const id = `${this.collectionData?.id}`;
      const body: IUpdateCollection = {
        name: this.collectionData.name,
        description: this.collectionData.description,
        avatarUrl: this.collectionData.avatarUrl,
        projectId: this.collectionData.projectId,
        smc: {
          contractAddress: this.f.contractAddress.value,
          contractType: this.f.contractType.value,
          networkID: Number(this.f.networkID.value),
          tokenSymbol: this.f.tokenSymbol.value,
        }
      };
      this.layergPortalService.updateCollections(id, body).subscribe((d) => {
        this.activeModal.dismiss();
        this.linkContractUpdated.emit();
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
  // handleDetail(): void {
  //   try {
  //     this.error = '';
  //     const id = `${this.collectionData?.id}/${this.collectibleData?.id}`;
  //     this.layergPortalService.detailCollectible(id).subscribe((d) => {
  //       this.f.name.setValue(d.name);
  //       this.f.description.setValue(d.description);
  //       this.f.tokenId.setValue(d.tokenId);
  //       this.f.quantity.setValue(d.quantity);
  //       this.imagePreview = d.media.S3Url || d.media.IPFSUrl;
  //       this.f.avatarUrl.setValue(d.media.S3Url || d.media.IPFSUrl);
  //       // Clear the existing attributes
  //       this.attributes.clear();
  //
  //       // Map the attributes from the data
  //       d.metadata.metadata.attributes.forEach((attr: { trait_type: string; value: string }) => {
  //         this.attributes.push(this.formBuilder.group({
  //           trait_type: [attr.trait_type, Validators.required],
  //           value: [attr.value, Validators.required],
  //         }));
  //       });
  //       this.item = d;
  //     }, err => {
  //       console.log(err);
  //       this.error = err;
  //     });
  //   } catch (e) {
  //     console.log(e);
  //     this.error = e;
  //   } finally {
  //   }
  // }
}
