import type { SVGProps } from "preact/compat";

function Isotipo(props: SVGProps<SVGSVGElement>) {
	return (
		<svg
			width={50}
			xmlns="http://www.w3.org/2000/svg"
			xmlSpace="preserve"
			viewBox="-9.22 -9.13 146.78 216.78"
			role="img"
			aria-label="Jota"
			className="fill-none stroke-current [stroke-linecap:round] [stroke-linejoin:round] [stroke-miterlimit:1.5]"
			style={{
				"--stroke-width": `${props.strokeWidth ?? 13}px`,
			}}
			{...props}
		>
			<path
				d="m34.641 25.37.093.026"
				className="fill-none stroke-current"
				style={{
					strokeWidth: "var(--stroke-width, 13px)",
				}}
				transform="rotate(205.006 52.474 25.253)"
			/>

			<path
				fill="none"
				d="m51.051 48.066 15.265-9.311 1.855 23.084M7.539 157.26s8.85-5.286 17.407-23.854c2.29-4.969 5.193-8.964 8.319-12.174a39.2 39.2 0 0 1 38.496-10.468c8.548 2.38 18.347 7.133 21.416 16.625 5.802 17.944-11.927 29.871-11.927 29.871s-16.956 10.745-39.261 11.819c-22.304 1.075-32.09-8.058-34.45-11.819"
				className="fill-transparent stroke-current"
				style={{
					strokeWidth: "var(--stroke-width, 13px)",
				}}
				transform="translate(-1.04 1.271)"
			/>

			<path
				fill="none"
				d="M94.343 134.364V60.988a11.12 11.12 0 0 0-8.012-10.677c-3.915-1.14-8.965-.457-14.458 5.692-12.357 13.834 0 35.056 0 35.056S37.665 88.25 50.941 48.066c0 0-7.005-16.814 4.277-30.782C62.334 8.473 74.222 6.021 82.03 5.412a39.1 39.1 0 0 1 19.329 3.476 43.6 43.6 0 0 1 5.75 3.175 35 35 0 0 1 15.773 29.246l.001 94.05s1.469 36.584-32.158 48.371c-31.269 10.96-40.429 5.851-41.56 5.081a1 1 0 0 0-.176-.115c-.807-.435-11.169-6.283-5.406-19.084l.358-.652"
				className="fill-transparent stroke-current"
				style={{
					strokeWidth: "var(--stroke-width, 13px)",
				}}
				transform="translate(-1.04 1.271)"
			/>
		</svg>
	);
}

export default Isotipo;
