import { expect, mock, test } from "bun:test";

// The store is English-only in the test: the i18n catalogs load through
// `import.meta.glob`, which Bun does not implement, and translating a message is
// not what this checks.
mock.module("@/lib/shared/i18n/errors", () => ({
	translateError: (error: unknown) => String(error),
}));

const { notify, removeError, roomErrors } = await import(
	"@/lib/shared/views/stores/errors"
);

test("the stack keeps the newest three cards", () => {
	roomErrors.value = [];
	notify("one");
	notify("two");
	notify("three");
	notify("four");

	const shown = roomErrors.value.filter((e) => !e.leaving);
	expect(shown.map((e) => e.message)).toEqual(["two", "three", "four"]);
});

test("a dismissed card starts leaving once and is gone after the exit", async () => {
	roomErrors.value = [];
	notify("copied");
	const [card] = roomErrors.value;

	removeError(card.id);
	removeError(card.id);
	expect(roomErrors.value[0].leaving).toBe(true);

	await new Promise((resolve) => setTimeout(resolve, 220));
	expect(roomErrors.value).toHaveLength(0);
});
