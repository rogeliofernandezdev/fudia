export async function uploadProductImage(productId:string,file:File){
  const form=new FormData();
  form.append("file",file);
  const response=await fetch(`/api/admin/products/${productId}/image`,{method:"POST",body:form});
  if(!response.ok){
    const body=await response.json().catch(()=>({}));
    throw new Error(body.message??"No pudimos subir la imagen del producto.");
  }
  return response.json() as Promise<{imageUrl:string}>;
}
