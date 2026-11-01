import type {TipPolicy,TipSettings} from "./types";

// Editable draft of a policy: suggesting a tip always shows the suggestion.
export function tipDraftFromPolicy(policy:TipPolicy):TipSettings{
 return {mode:policy.mode,percentage:policy.percentage,showSuggestion:policy.mode==="suggest"||policy.showSuggestion,allowFixed:policy.allowFixed,channels:policy.channels};
}
