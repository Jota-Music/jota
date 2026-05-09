import ClearLayout from "@/lib/shared/views/ui/layouts/clear"

function RegisterPage() {
    return (
        <ClearLayout className="gap-4">
            <form>
                <input type="text" name="spotify" />
            </form>
        </ClearLayout>
    )
}

export default RegisterPage