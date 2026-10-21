import assert from "node:assert/strict";
import {readFileSync} from "node:fs";
import test from "node:test";

const read=file=>readFileSync(new URL(`../${file}`,import.meta.url),"utf8");
const page=read("src/modules/platform/presentation/platform-companies-page.tsx");

test("empresas: catálogos de plan y situación de pago provienen del API",()=>{
 assert.match(page,/data\?\.planOptions\.map/);
 assert.match(page,/data\?\.standingOptions\.map/);
 assert.match(page,/standingOptions\.find/);
 assert.doesNotMatch(page,/<option value="(up_to_date|due_soon|overdue|trial|cancelled|none)"/);
 assert.doesNotMatch(page,/<option value="[0-9a-f-]{36}"/);
});

test("empresas: datos remotos presentan skeleton, error con reintento y vacío",()=>{
 assert.match(page,/CompaniesTableSkeleton/);
 assert.match(page,/aria-busy="true"/);
 assert.match(page,/No pudimos cargar las empresas/);
 assert.match(page,/Reintentar/);
 assert.match(page,/Aún no hay empresas/);
 assert.match(page,/Sin coincidencias/);
});

test("empresas: cada empresa abre su propio formulario sin cambiar la sesión",()=>{
 assert.match(page,/href=\{companySettingsPath\(company\.id\)\}/);
 assert.doesNotMatch(page,/switchContext/);
 const detail=read("src/modules/platform/presentation/platform-company-detail-page.tsx");
 assert.match(detail,/getOrganizationSubscription\(organizationId\)/);
 assert.match(detail,/<SubscriptionWorkspace[^>]*organizationId=\{organizationId\}/);
 assert.match(detail,/PlatformSubscriptionSkeleton/);
 assert.match(detail,/Reintentar/);
 assert.match(detail,/Sin suscripción/);
 const api=read("src/modules/platform/infrastructure/platform-api.ts");
 assert.match(api,/organizations\/\$\{encodeURIComponent\(organizationId\)\}\/subscription/);
 const workspace=read("src/modules/platform/presentation/platform-subscription-page.tsx");
 assert.match(workspace,/changeOrganizationSubscription\(draft,organizationId\)/);
 assert.match(workspace,/\},organizationId\)/);
 assert.match(read("src/app/(admin)/settings/companies/[id]/page.tsx"),/<PlatformCompanyDetailPage organizationId=\{id\}\/>/);
});

test("empresas: página independiente del admin, solo para el administrador de plataforma",()=>{
 assert.match(read("src/shared/routing/page-routes.ts"),/companiesSettings:"\/settings\/companies"/);
 assert.match(read("src/app/(admin)/settings/companies/page.tsx"),/<PlatformCompaniesPage\/>/);
 assert.ok(read("src/shell/navigation.ts").includes('href:pageRoutes.companiesSettings,name:"Empresas y planes",icon:"contract",module:"locales",platformAdminOnly:true'),"El menú principal expone la opción solo al administrador de plataforma");
 assert.doesNotMatch(read("src/modules/platform/presentation/platform-shell.tsx"),/Empresas y planes|companies/i,"No vive dentro del panel de Plataforma");
 assert.match(page,/if\(!user\?\.platformAdmin\)return <RestrictedCompanies\/>/);
 assert.match(read("src/modules/platform/presentation/platform-company-detail-page.tsx"),/if\(!user\?\.platformAdmin\)return <RestrictedCompanies\/>/);
 assert.match(read("src/modules/platform/presentation/platform-skeletons.css"),/\.platform-skeleton-block,\.platform-skeleton-copy i\{/);
 const proxy=read("src/app/api/platform/organizations/route.ts");
 assert.match(proxy,/export const GET=proxy/);
 assert.match(proxy,/export const POST=proxy/);
 assert.match(proxy,/searchParams\.forEach/);
});

test("empresas: ver detalle abre la ficha de solo lectura con X como único cierre",()=>{
 const dialog=read("src/modules/platform/presentation/platform-company-detail-dialog.tsx");
 assert.match(page,/<RowActionButton action="view" label="Ver detalle" onClick=\{\(\)=>setDetailId\(company\.id\)\}\/>/);
 assert.match(dialog,/getPlatformOrganization\(organizationId\)/);
 assert.match(dialog,/aria-label="Cerrar detalle"/);
 assert.doesNotMatch(dialog,/>Cerrar</,"El detalle no agrega botón Cerrar en el pie");
 assert.match(dialog,/CompanyDetailSkeleton/);
 assert.match(dialog,/No pudimos cargar el detalle/);
 for(const section of ["Datos generales y fiscales","Suscripción y uso","Administradores","Locales"])assert.ok(dialog.includes(section),section);
 assert.match(dialog,/Intl\.DisplayNames/,"El país se resuelve sin catálogo propio");
});
