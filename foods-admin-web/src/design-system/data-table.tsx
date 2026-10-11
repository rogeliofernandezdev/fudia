import type {ComponentPropsWithoutRef,ReactNode} from "react";

type DataTableProps = Omit<ComponentPropsWithoutRef<"table">,"children"> & {
  children: ReactNode;
  label: string;
  containerClassName?: string;
};

/** Shared native table: accessible horizontal scrolling without replacing rows with cards. */
export function DataTable({label,containerClassName="",children,...tableProps}:DataTableProps){
  return (
    <div className={`table-wrap min-w-0 max-w-full ${containerClassName}`.trim()} role="region" aria-label={label} tabIndex={0}>
      <table {...tableProps}>{children}</table>
    </div>
  );
}
