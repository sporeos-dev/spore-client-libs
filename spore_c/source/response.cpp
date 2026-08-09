#include "response.h"

namespace spore
{
    std::string_view ResponseError::getCode() const { return code; }
    std::string_view ResponseError::getWhat() const { return what; }
    void ResponseError::setCode(std::string_view v) { code = v; }
    void ResponseError::setWhat(std::string_view v) { what = v; }

}  // namespace spore
