export type ExpenseStatus = 'ACTIVE' | 'VOID';

export interface Expense {
  id: string;
  categoryId: string;
  description: string;
  amount: number;
  paymentMethod: string;
  status: ExpenseStatus;
  createdAt: string;
  createdBy?: string;
}

export interface ExpenseCategory {
  id: string;
  name: string;
  active: boolean;
}
