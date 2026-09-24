import {apiFetch} from "@/shared/api/client";
import type {Expense,ExpenseCategoriesResponse,ExpenseCategory,ExpenseDraft,ExpensesResponse} from "../domain/expense";
export type ExpensesQuery={q:string;categoryId:string;status:string;from:string;to:string;page:number;pageSize:number};
export function listExpenses(query:ExpensesQuery){
 const params=new URLSearchParams({q:query.q,categoryId:query.categoryId,status:query.status,from:query.from,to:query.to,page:String(query.page),pageSize:String(query.pageSize)});
 return apiFetch<ExpensesResponse>("expenses?"+params.toString());
}
export function createExpense(draft:ExpenseDraft){return apiFetch<Expense>("expenses",{method:"POST",body:JSON.stringify(draft)})}
export function voidExpense(id:string,reason:string){return apiFetch<Expense>("expenses/"+id+"/void",{method:"POST",body:JSON.stringify({reason})})}
export function listExpenseCategories(input:{q?:string;status?:string;page?:number;pageSize?:number}={}){
 const params=new URLSearchParams({q:input.q??"",status:input.status??"",page:String(input.page??1),pageSize:String(input.pageSize??20)});
 return apiFetch<ExpenseCategoriesResponse>("expense-categories?"+params.toString());
}
export function saveExpenseCategory(input:{id?:string;name:string}){return apiFetch<ExpenseCategory>(input.id?"expense-categories/"+input.id:"expense-categories",{method:input.id?"PATCH":"POST",body:JSON.stringify({name:input.name.trim()})})}
export function setExpenseCategoryActive(id:string,active:boolean){return apiFetch<void>("expense-categories/"+id+"/status",{method:"PATCH",body:JSON.stringify({active})})}
