package app

import "testing"

func TestHumanAction(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		string(EventTrackerCreateReminder): "создано напоминание",
		string(EventTrackerCreateTask):     "создана задача",
		string(EventFinanceCreateTxn):      "зарегистрирована транзакция",
		string(EventKnowledgeSaveQuery):    "запрос сохранён в knowledge",
		string(EventKnowledgeSaveNote):     "заметка сохранена в knowledge",
		"":                                 "действие выполнено",
		"v1.custom.action":                 "v1.custom.action",
	}

	for input, want := range cases {
		if got := HumanAction(input); got != want {
			t.Fatalf("HumanAction(%q) = %q, want %q", input, got, want)
		}
	}
}
