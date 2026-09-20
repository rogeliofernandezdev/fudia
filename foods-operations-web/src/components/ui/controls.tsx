import type { ComponentProps, ReactNode } from "react";
import Link from "next/link";
import styles from "./controls.module.css";

type Tone = "neutral" | "primary" | "operational" | "danger";
type ButtonProps = ComponentProps<"button"> & { tone?: Tone; layout?: "action" | "icon" | "card" };

export function Button({ className = "", type = "button", tone = "neutral", layout = "action", ...props }: ButtonProps) {
  return <button {...props} type={type} data-ui="button" data-tone={tone} data-layout={layout} className={`${styles.button} ${className}`}/>;
}

export function TableAction({ label, className = "", type = "button", ...props }: Omit<ButtonProps, "layout" | "aria-label"> & { label: string }) {
  return <Button {...props} type={type} layout="icon" aria-label={label} data-tooltip={label} className={`${styles.tableAction} ${className}`}/>;
}

export function ActionLink({ className = "", tone = "operational", ...props }: ComponentProps<typeof Link> & { tone?: Tone }) {
  return <Link {...props} data-ui="button" data-tone={tone} data-layout="action" className={`${styles.button} ${className}`}/>;
}

export function Input({ className = "", type = "text", ...props }: ComponentProps<"input">) {
  return <input {...props} type={type} data-ui="input" className={`${styles.input} ${className}`}/>;
}

export function Textarea({ className = "", ...props }: ComponentProps<"textarea">) {
  return <textarea {...props} data-ui="textarea" className={`${styles.textarea} ${className}`}/>;
}

export function Select({ className = "", children, ...props }: ComponentProps<"select">) {
  return <select {...props} data-ui="select" className={`${styles.select} ${className}`}>{children}</select>;
}

export function Label({ className = "", ...props }: ComponentProps<"label">) {
  return <label {...props} data-ui="label" className={`${styles.label} ${className}`}/>;
}

export function Field({ id, label, hint, error, children }: { id: string; label: string; hint?: string; error?: string; children: ReactNode }) {
  return <div className={styles.field}>
    <Label htmlFor={id}>{label}</Label>{children}
    {(hint || error) && <small id={`${id}-help`} role={error ? "alert" : undefined} className={error ? styles.error : styles.hint}>{error || hint}</small>}
  </div>;
}

export function Table({ children, caption }: { children: ReactNode; caption: string }) {
  return <div className={styles.tableScroll} tabIndex={0} role="region" aria-label={caption}>
    <table className={styles.table}><caption>{caption}</caption>{children}</table>
  </div>;
}
