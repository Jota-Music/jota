import { expect, test } from "bun:test";
import { secondsToTime } from "@/lib/shared/utils/format";

test("secondsToTime pads minutes and seconds", () => {
	expect(secondsToTime(0)).toBe("00:00");
	expect(secondsToTime(9)).toBe("00:09");
	expect(secondsToTime(59)).toBe("00:59");
	expect(secondsToTime(60)).toBe("01:00");
	expect(secondsToTime(3599)).toBe("59:59");
});

test("secondsToTime keeps minute count beyond 60 minutes", () => {
	expect(secondsToTime(3661)).toBe("61:01");
});

test("secondsToTime floors fractional seconds", () => {
	expect(secondsToTime(83.7)).toBe("01:23");
});
