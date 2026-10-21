package httpapi

import "testing"

func TestModuleCatalogMatchesAdminNavigation(t *testing.T) {
	want := map[string]struct{
		name string
		icon string
		descriptionContains string
	}{
		"productos": {name:"Carta y productos",icon:"utensils",descriptionContains:"disponibilidad"},
		"recetas": {name:"Recetas",icon:"cookingPot"},
		"kardex": {name:"Kardex",icon:"ledger"},
		"usuarios": {name:"Usuarios y roles",icon:"users"},
	}
	for key,expected:=range want {
		found:=false
		for _,module:=range moduleCatalog {
			if module.Key!=key { continue }
			found=true
			if module.Name!=expected.name {
				t.Fatalf("%s name: expected %q, got %q",key,expected.name,module.Name)
			}
			if module.Icon!=expected.icon {
				t.Fatalf("%s icon: expected %q, got %q",key,expected.icon,module.Icon)
			}
			if expected.descriptionContains!="" && !containsFold(module.Description,expected.descriptionContains) {
				t.Fatalf("%s description must mention %q, got %q",key,expected.descriptionContains,module.Description)
			}
			break
		}
		if !found { t.Fatalf("module %s not found",key) }
	}
}

func containsFold(value,needle string) bool {
	if len(needle)==0 { return true }
	for i:=0;i+len(needle)<=len(value);i++ {
		match:=true
		for j:=0;j<len(needle);j++ {
			a:=value[i+j]
			b:=needle[j]
			if a>='A'&&a<='Z' { a=a-'A'+'a' }
			if b>='A'&&b<='Z' { b=b-'A'+'a' }
			if a!=b { match=false;break }
		}
		if match { return true }
	}
	return false
}
