import Register from "@/lib/auth/views/ui/register"
import ClearLayout from "@/lib/shared/views/ui/layouts/clear"

function RegisterPage() {
    return (
        <ClearLayout className="gap-4">
            <Register />
        </ClearLayout>
    )
}

export default RegisterPage