import {NgModule} from '@angular/core';
import {Routes, RouterModule} from '@angular/router';
import {LoginComponent, LoginGuard} from './login/login.component';
import {AuthenticationGuard} from './authentication.guard';
import {BaseComponent, PageviewGuard} from './base/base.component';
import {GraphInitNodesResolver, StatusComponent} from './status/status.component';
import {ConfigComponent, ConfigResolver} from './config/config.component';
import {UsersComponent, UsersResolver} from './users/users.component';
import {RuntimeComponent, RuntimeResolver} from './runtime/runtime.component';
import {StorageCollectionResolver, StorageListComponent, StorageSearchResolver} from './storage/storage.component';
import {StorageObjectComponent, StorageObjectResolver} from './storage-object/storage-object.component';
import {LeaderboardsComponent, LeaderboardListResolver} from './leaderboards/leaderboards.component';
import {AccountListComponent, AccountSearchResolver} from './accounts/accounts.component';
import {AccountComponent, AccountResolver} from './account/account.component';
import {ProfileComponent} from './account/profile/profile.component';
import {AuthenticationComponent} from './account/authentication/authentication.component';
import {WalletComponent, WalletLedgerResolver} from './account/wallet/wallet.component';
import {FriendsComponent, FriendsResolver} from './account/friends/friends.component';
import {GroupsComponent, GroupsResolver} from './account/groups/groups.component';
import {GroupDetailsComponent} from './group/details/groupDetailsComponent';
import {GroupMembersComponent, GroupMembersResolver} from './group/members/groupMembers.component';
import {MatchesComponent, MatchesResolver, NodesResolver} from './matches/matches.component';
import {GroupListComponent, GroupSearchResolver} from './groups/groups.component';
import {GroupComponent, GroupResolver} from './group/group.component';
import {LeaderboardComponent, LeaderboardResolver} from './leaderboard/leaderboard.component';
import {LeaderboardDetailsComponent} from './leaderboard/details/details.component';
import {LeaderboardRecordsComponent, LeaderboardRecordsResolver} from './leaderboard/records/records.component';
import {ApiExplorerComponent, ApiExplorerEndpointsResolver} from './apiexplorer/apiexplorer.component';
import {PurchasesComponent, PurchasesResolver} from './account/purchases/purchases.component';
import {ChatListComponent, ChatSearchResolver} from './channels/chat-list.component';
import {SubscriptionsComponent, SubscriptionsResolver} from './account/subscriptions/subscriptions.component';
import {PurchasesListComponent} from './purchases/purchases-list.component';
import {SubscriptionsListComponent} from './subscriptions/subscriptions-list.component';
import {CollectionsComponent, CollectionsResolver} from './collections/collections.component';
import {CreateCollectionComponent, CreateCollectionResolver} from './collections/create/createCollection.component';
import {CollectionDetailComponent} from './collections/detail/collectionDetail.component';
import {CollectionComponent, CollectionResolver} from './collection/collection.component';
import {CollectionDetail1Component} from './collection/detail/collectionDetail.component';
import {NotificationsListComponent} from './notifications/notifications-list.component';
import {NotificationsComponent, NotificationsResolver} from "./account/notifications/notifications.component";

const routes: Routes = [
  {
    path: '',
    component: BaseComponent,
    canActivate: [AuthenticationGuard],
    canActivateChild: [PageviewGuard],
    children: [
      {path: '', redirectTo: 'status', pathMatch: 'full'},
      {path: 'status', component: StatusComponent, resolve: [GraphInitNodesResolver]},
      {path: 'config', component: ConfigComponent, resolve: [ConfigResolver]},
      {path: 'users', component: UsersComponent, resolve: [UsersResolver]},
      {path: 'modules', component: RuntimeComponent, resolve: [RuntimeResolver]},
      {path: 'collections', component: CollectionsComponent, resolve: [CollectionsResolver],},
      {path: 'collections/create', component: CreateCollectionComponent, resolve: [CreateCollectionResolver]},
      {path: 'collections/:id', component: CollectionComponent, resolve: [CollectionResolver],
        children: [
          {path: '', redirectTo: 'details', pathMatch: 'full'},
          {path: 'details', component: CollectionDetail1Component, resolve: []},
        ]
      },
      {path: 'storage', component: StorageListComponent, resolve: [StorageCollectionResolver, StorageSearchResolver], pathMatch: 'full'},
      {path: 'storage/:collection/:key/:user_id', component: StorageObjectComponent, resolve: [StorageObjectResolver], pathMatch: 'full'},
      {path: 'leaderboards', component: LeaderboardsComponent, resolve: [LeaderboardListResolver]},
      {path: 'leaderboards/:id', component: LeaderboardComponent, resolve: [LeaderboardResolver],
        children: [
          {path: '', redirectTo: 'details', pathMatch: 'full'},
          {path: 'details', component: LeaderboardDetailsComponent, resolve: []},
          {path: 'records', component: LeaderboardRecordsComponent, resolve: [LeaderboardRecordsResolver]},
        ]
      },
      {path: 'matches', component: MatchesComponent, resolve: [MatchesResolver, NodesResolver]},
      {path: 'groups', component: GroupListComponent, resolve: [GroupSearchResolver]},
      {
        path: 'groups/:id', component: GroupComponent, resolve: [GroupResolver],
        children: [
          {path: '', redirectTo: 'details', pathMatch: 'full'},
          {path: 'details', component: GroupDetailsComponent, resolve: []},
          {path: 'members', component: GroupMembersComponent, resolve: [GroupMembersResolver], runGuardsAndResolvers: 'always'},
        ]
      },
      {path: 'accounts', component: AccountListComponent, resolve: [AccountSearchResolver]},
      {
        path: 'accounts/:id', component: AccountComponent, resolve: [AccountResolver],
        children: [
          {path: '', redirectTo: 'profile', pathMatch: 'full'},
          {path: 'profile', component: ProfileComponent, resolve: []},
          {path: 'authentication', component: AuthenticationComponent, resolve: []},
          {path: 'wallet', component: WalletComponent, resolve: [WalletLedgerResolver]},
          {path: 'friends', component: FriendsComponent, resolve: [FriendsResolver]},
          {path: 'groups', component: GroupsComponent, resolve: [GroupsResolver]},
          {path: 'notifications', component: NotificationsComponent, resolve: [NotificationsResolver]},
          {path: 'purchases', component: PurchasesComponent, resolve: [PurchasesResolver]},
          {path: 'subscriptions', component: SubscriptionsComponent, resolve: [SubscriptionsResolver]}
        ]
      },
      {path: 'apiexplorer', component: ApiExplorerComponent, resolve: [ApiExplorerEndpointsResolver]},
      {path: 'chat', component: ChatListComponent, resolve: [ChatSearchResolver]},
      {path: 'notifications', component: NotificationsListComponent, resolve: [NotificationsResolver]},
      {path: 'purchases', component: PurchasesListComponent, resolve: [PurchasesResolver]},
      {path: 'subscriptions', component: SubscriptionsListComponent, resolve: [SubscriptionsResolver]},
    ]},
  {path: 'login', component: LoginComponent, canActivate: [LoginGuard]},

  // Fallback redirect.
  {path: '**', redirectTo: ''}
];

@NgModule({
  imports: [
    RouterModule.forRoot(routes, {useHash: true}),
    // RouterModule.forRoot(routes, { useHash: true, enableTracing: true }), // TODO debugging purposes only
  ],
  exports: [RouterModule]
})
export class AppRoutingModule { }
