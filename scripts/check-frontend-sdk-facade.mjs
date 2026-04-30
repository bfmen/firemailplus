#!/usr/bin/env node
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';

const apiPath = resolve(process.cwd(), 'frontend/src/lib/api.ts');
const source = readFileSync(apiPath, 'utf8');

const allowedDirectUnstableEndpoints = new Set([
  '/emails/stats',
]);

const directEndpointMatches = [
  ...source.matchAll(/this\.request\((['"`])([^'"`]+)\1/g),
].map((match) => match[2]);

const unexpectedDirect = directEndpointMatches.filter(
  (endpoint) => endpoint.startsWith('/') && !allowedDirectUnstableEndpoints.has(endpoint)
);

if (unexpectedDirect.length > 0) {
  console.error('Stable facade methods must use generated OpenAPI URL helpers:');
  for (const endpoint of unexpectedDirect) console.error(`  - ${endpoint}`);
  process.exit(1);
}

const requiredHelpers = [
  'getLoginUrl',
  'getLogoutUrl',
  'getGetCurrentUserUrl',
  'getListEmailAccountsUrl',
  'getGetEmailAccountUrl',
  'getCreateEmailAccountUrl',
  'getUpdateEmailAccountUrl',
  'getDeleteEmailAccountUrl',
  'getTestEmailAccountUrl',
  'getSyncEmailAccountUrl',
  'getMarkAccountAsReadUrl',
  'getBatchDeleteEmailAccountsUrl',
  'getBatchSyncEmailAccountsUrl',
  'getBatchMarkAccountsAsReadUrl',
  'getGetAccountJobStatusUrl',
  'getListEmailGroupsUrl',
  'getCreateEmailGroupUrl',
  'getUpdateEmailGroupUrl',
  'getDeleteEmailGroupUrl',
  'getSetDefaultEmailGroupUrl',
  'getReorderEmailGroupsUrl',
  'getInitGmailOAuthUrl',
  'getInitOutlookOAuthUrl',
  'getHandleOAuthCallbackUrl',
  'getCreateOAuthAccountUrl',
  'getCreateManualOAuthAccountUrl',
  'getCreateCustomEmailAccountUrl',
  'getListEmailsUrl',
  'getSearchEmailsUrl',
  'getListFoldersUrl',
  'getGetFolderUrl',
  'getBatchEmailOperationsUrl',
  'getMarkFolderAsReadUrl',
  'getMoveEmailUrl',
  'getCreateFolderUrl',
  'getUpdateFolderUrl',
  'getDeleteFolderUrl',
  'getSyncFolderUrl',
  'getGetEmailUrl',
  'getMarkEmailAsReadUrl',
  'getMarkEmailAsUnreadUrl',
  'getToggleEmailStarUrl',
  'getDeleteEmailUrl',
  'getUpdateEmailUrl',
  'getDownloadAttachmentUrl',
  'getSendEmailUrl',
  'getSendBulkEmailsUrl',
  'getGetSendStatusUrl',
  'getResendEmailUrl',
  'getSaveDraftUrl',
  'getGetDraftUrl',
  'getUpdateDraftUrl',
  'getDeleteDraftUrl',
  'getListDraftsUrl',
  'getCreateTemplateUrl',
  'getGetTemplateUrl',
  'getUpdateTemplateUrl',
  'getDeleteTemplateUrl',
  'getListTemplatesUrl',
  'getReplyEmailUrl',
  'getReplyAllEmailUrl',
  'getForwardEmailUrl',
  'getArchiveEmailUrl',
];

const missingHelpers = requiredHelpers.filter((helper) => !source.includes(`${helper}(`));
if (missingHelpers.length > 0) {
  console.error('Facade is missing generated helper usage:');
  for (const helper of missingHelpers) console.error(`  - ${helper}`);
  process.exit(1);
}

console.log(
  `Frontend SDK facade check passed with ${requiredHelpers.length} generated helper mappings.`
);
